package collab

import (
	"context"
	"encoding/json"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/property"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/samber/lo"
)

type applyAddPropertyItem struct {
	Kind            string          `json:"kind"`
	SceneID         string          `json:"sceneId"`
	PropertyID      string          `json:"propertyId"`
	SchemaGroupID   string          `json:"schemaGroupId"`
	Index           *int            `json:"index,omitempty"`
	NameFieldType   *string         `json:"nameFieldType,omitempty"`
	NameFieldValue  json.RawMessage `json:"nameFieldValue,omitempty"`
	BaseSceneRev    *int64          `json:"baseSceneRev,omitempty"`
}

type applyRemovePropertyItem struct {
	Kind          string `json:"kind"`
	SceneID       string `json:"sceneId"`
	PropertyID    string `json:"propertyId"`
	SchemaGroupID string `json:"schemaGroupId"`
	ItemID        string `json:"itemId"`
	BaseSceneRev  *int64 `json:"baseSceneRev,omitempty"`
}

type applyMovePropertyItem struct {
	Kind          string `json:"kind"`
	SceneID       string `json:"sceneId"`
	PropertyID    string `json:"propertyId"`
	SchemaGroupID string `json:"schemaGroupId"`
	ItemID        string `json:"itemId"`
	Index         int    `json:"index"`
	BaseSceneRev  *int64 `json:"baseSceneRev,omitempty"`
}

func reloadSceneAfterProperty(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID) (*scene.Scene, error) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, errSceneReloadFailed
	}
	return scenes[0], nil
}

// applyAddPropertyItemOp runs Property.AddItem (list schema groups).
func applyAddPropertyItemOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddPropertyItem
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}
	if !sceneMustNotBeLockedByPeer(ctx, hub, from, sid) {
		return nil
	}
	pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	if fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from) == nil {
		return nil
	}
	var nameVal *property.Value
	if p.NameFieldType != nil && *p.NameFieldType != "" && len(p.NameFieldValue) > 0 && string(p.NameFieldValue) != "null" {
		var valIface interface{}
		if err := json.Unmarshal(p.NameFieldValue, &valIface); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
			return nil
		}
		vt := gqlmodel.ValueType(*p.NameFieldType)
		nameVal = gqlmodel.FromPropertyValueAndType(valIface, vt)
		if nameVal == nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid name field value"}})
			return nil
		}
	}
	sg := gqlmodel.ID(p.SchemaGroupID)
	ptr := gqlmodel.FromPointer(gqlmodel.ToStringIDRef[id.PropertySchemaGroup](&sg), nil, nil)
	if ptr == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid schema group"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, err = uc.Property.AddItem(opCtx, interfaces.AddPropertyItemParam{
		PropertyID:     pid,
		Pointer:        ptr,
		Index:          p.Index,
		NameFieldValue: nameVal,
	}, op)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	sc, err2 := reloadSceneAfterProperty(ctx, uc, op, sid)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	extra := map[string]any{
		"sceneId":       p.SceneID,
		"propertyId":    p.PropertyID,
		"schemaGroupId": p.SchemaGroupID,
	}
	if hub != nil {
		extra["propertyDocClock"] = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	broadcastApplied(ctx, hub, from, "add_property_item", extra, sc)
	return nil
}

func applyRemovePropertyItemOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemovePropertyItem
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}
	if !sceneMustNotBeLockedByPeer(ctx, hub, from, sid) {
		return nil
	}
	pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	if fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from) == nil {
		return nil
	}
	itemID := gqlmodel.ID(p.ItemID)
	ptr := gqlmodel.FromPointer(
		lo.ToPtr(id.PropertySchemaGroupID(p.SchemaGroupID)),
		&itemID,
		nil,
	)
	if ptr == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid pointer"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err = uc.Property.RemoveItem(opCtx, interfaces.RemovePropertyItemParam{
		PropertyID: pid,
		Pointer:    ptr,
	}, op)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	sc, err2 := reloadSceneAfterProperty(ctx, uc, op, sid)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	extra := map[string]any{
		"sceneId":       p.SceneID,
		"propertyId":    p.PropertyID,
		"schemaGroupId": p.SchemaGroupID,
		"itemId":        p.ItemID,
	}
	if hub != nil {
		extra["propertyDocClock"] = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	broadcastApplied(ctx, hub, from, "remove_property_item", extra, sc)
	return nil
}

func applyMovePropertyItemOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyMovePropertyItem
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}
	if !sceneMustNotBeLockedByPeer(ctx, hub, from, sid) {
		return nil
	}
	pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	if fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from) == nil {
		return nil
	}
	itemID := gqlmodel.ID(p.ItemID)
	ptr := gqlmodel.FromPointer(
		lo.ToPtr(id.PropertySchemaGroupID(p.SchemaGroupID)),
		&itemID,
		nil,
	)
	if ptr == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid pointer"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, err = uc.Property.MoveItem(opCtx, interfaces.MovePropertyItemParam{
		PropertyID: pid,
		Pointer:    ptr,
		Index:      p.Index,
	}, op)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	sc, err2 := reloadSceneAfterProperty(ctx, uc, op, sid)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	extra := map[string]any{
		"sceneId":       p.SceneID,
		"propertyId":    p.PropertyID,
		"schemaGroupId": p.SchemaGroupID,
		"itemId":        p.ItemID,
		"index":         p.Index,
	}
	if hub != nil {
		extra["propertyDocClock"] = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	broadcastApplied(ctx, hub, from, "move_property_item", extra, sc)
	return nil
}
