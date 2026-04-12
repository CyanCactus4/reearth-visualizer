package collab

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/property"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
	"github.com/samber/lo"
)

type applyAddPropertyItem struct {
	Kind           string          `json:"kind"`
	SceneID        string          `json:"sceneId"`
	PropertyID     string          `json:"propertyId"`
	SchemaGroupID  string          `json:"schemaGroupId"`
	Index          *int            `json:"index,omitempty"`
	NameFieldType  *string         `json:"nameFieldType,omitempty"`
	NameFieldValue json.RawMessage `json:"nameFieldValue,omitempty"`
	BaseSceneRev   *int64          `json:"baseSceneRev,omitempty"`
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

// groupListItemIndex returns the index of a group inside a PropertyGroupList for the given schema group.
func groupListItemIndex(prop *property.Property, schemaGroupID id.PropertySchemaGroupID, itemID id.PropertyItemID) (int, bool) {
	if prop == nil {
		return 0, false
	}
	for _, it := range prop.Items() {
		gl, ok := it.(*property.GroupList)
		if !ok || gl.SchemaGroup() != schemaGroupID {
			continue
		}
		for i, g := range gl.Groups() {
			if g != nil && g.ID() == itemID {
				return i, true
			}
		}
	}
	return 0, false
}

func runAddPropertyItemCore(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID, p *applyAddPropertyItem) (*scene.Scene, *property.Group, error) {
	var nameVal *property.Value
	if p.NameFieldType != nil && *p.NameFieldType != "" && len(p.NameFieldValue) > 0 && string(p.NameFieldValue) != "null" {
		var valIface interface{}
		if err := json.Unmarshal(p.NameFieldValue, &valIface); err != nil {
			return nil, nil, err
		}
		vt := gqlmodel.ValueType(*p.NameFieldType)
		nameVal = gqlmodel.FromPropertyValueAndType(valIface, vt)
		if nameVal == nil {
			return nil, nil, errInvalidPropertyApply("invalid name field value")
		}
	}
	sg := gqlmodel.ID(p.SchemaGroupID)
	ptr := gqlmodel.FromPointer(gqlmodel.ToStringIDRef[id.PropertySchemaGroup](&sg), nil, nil)
	if ptr == nil {
		return nil, nil, errInvalidPropertyApply("invalid schema group")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, newGroup, err := uc.Property.AddItem(opCtx, interfaces.AddPropertyItemParam{
		PropertyID:     pid,
		Pointer:        ptr,
		Index:          p.Index,
		NameFieldValue: nameVal,
	}, op)
	if err != nil {
		return nil, nil, err
	}
	sc, err2 := reloadSceneAfterProperty(ctx, uc, op, sid)
	if err2 != nil {
		return nil, nil, err2
	}
	return sc, newGroup, nil
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
	sc, newGroup, err2 := runAddPropertyItemCore(ctx, uc, op, sid, pid, &p)
	if err2 != nil {
		code := "apply_failed"
		var inv errInvalidPropertyApply
		if errors.As(err2, &inv) {
			code = "invalid_payload"
		}
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": code, "message": err2.Error()}})
		return nil
	}
	if newGroup == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "add item produced no group"}})
		return nil
	}
	extra := map[string]any{
		"sceneId":       p.SceneID,
		"propertyId":    p.PropertyID,
		"schemaGroupId": p.SchemaGroupID,
		"itemId":        newGroup.ID().String(),
	}
	if hub != nil {
		extra["propertyDocClock"] = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	broadcastApplied(ctx, hub, from, "add_property_item", extra, sc)

	if hub != nil && hub.opStack != nil {
		inv := applyRemovePropertyItem{
			Kind:          "remove_property_item",
			SceneID:       p.SceneID,
			PropertyID:    p.PropertyID,
			SchemaGroupID: p.SchemaGroupID,
			ItemID:        newGroup.ID().String(),
		}
		invJSON, mErr := json.Marshal(&inv)
		if mErr == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_property_item",
				Forward:   d,
				Inverse:   invJSON,
			}
			go func() {
				pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
					log.Warnfc(pctx, "collab: undo stack: %v", err)
				}
			}()
		}
	}
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
	prop := fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from)
	if prop == nil {
		return nil
	}
	itemIid, err := id.PropertyItemIDFrom(p.ItemID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid itemId"}})
		return nil
	}
	sgID := id.PropertySchemaGroupID(p.SchemaGroupID)
	listIdx, ok := groupListItemIndex(prop, sgID, itemIid)
	if !ok {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "property item not found in list"}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		var bErr error
		invJSON, bErr = buildInverseAddPropertyItemJSONAfterRemove(ctx, uc, op, prop, &p, listIdx)
		if bErr != nil {
			log.Warnfc(ctx, "collab: remove_property_item undo snapshot: %v", bErr)
		}
	}
	sc, err2 := runRemovePropertyItemCore(ctx, uc, op, sid, pid, &p)
	if err2 != nil {
		code := "apply_failed"
		var inv errInvalidPropertyApply
		if errors.As(err2, &inv) {
			code = "invalid_payload"
		}
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": code, "message": err2.Error()}})
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

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "remove_property_item",
			Forward:   d,
			Inverse:   invJSON,
		}
		go func() {
			pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel2()
			if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
				log.Warnfc(pctx, "collab: undo stack: %v", err)
			}
		}()
	}
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
	prop := fetchPropertyForCollabApply(ctx, uc, op, sid, pid, from)
	if prop == nil {
		return nil
	}
	itemIid, err := id.PropertyItemIDFrom(p.ItemID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid itemId"}})
		return nil
	}
	oldIdx, ok := groupListItemIndex(prop, id.PropertySchemaGroupID(p.SchemaGroupID), itemIid)
	if !ok {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "property item not found in list"}})
		return nil
	}
	sc, err2 := runMovePropertyItemCore(ctx, uc, op, sid, pid, &p)
	if err2 != nil {
		code := "apply_failed"
		var inv errInvalidPropertyApply
		if errors.As(err2, &inv) {
			code = "invalid_payload"
		}
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": code, "message": err2.Error()}})
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

	if hub != nil && hub.opStack != nil && oldIdx != p.Index {
		inv := applyMovePropertyItem{
			Kind:          "move_property_item",
			SceneID:       p.SceneID,
			PropertyID:    p.PropertyID,
			SchemaGroupID: p.SchemaGroupID,
			ItemID:        p.ItemID,
			Index:         oldIdx,
		}
		invJSON, mErr := json.Marshal(&inv)
		if mErr == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "move_property_item",
				Forward:   d,
				Inverse:   invJSON,
			}
			go func() {
				pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
					log.Warnfc(pctx, "collab: undo stack: %v", err)
				}
			}()
		}
	}
	return nil
}

func runRemovePropertyItemCore(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID, p *applyRemovePropertyItem) (*scene.Scene, error) {
	itemID := gqlmodel.ID(p.ItemID)
	ptr := gqlmodel.FromPointer(
		lo.ToPtr(id.PropertySchemaGroupID(p.SchemaGroupID)),
		&itemID,
		nil,
	)
	if ptr == nil {
		return nil, errInvalidPropertyApply("invalid pointer")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err := uc.Property.RemoveItem(opCtx, interfaces.RemovePropertyItemParam{
		PropertyID: pid,
		Pointer:    ptr,
	}, op)
	if err != nil {
		return nil, err
	}
	return reloadSceneAfterProperty(ctx, uc, op, sid)
}

func runMovePropertyItemCore(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID, p *applyMovePropertyItem) (*scene.Scene, error) {
	itemID := gqlmodel.ID(p.ItemID)
	ptr := gqlmodel.FromPointer(
		lo.ToPtr(id.PropertySchemaGroupID(p.SchemaGroupID)),
		&itemID,
		nil,
	)
	if ptr == nil {
		return nil, errInvalidPropertyApply("invalid pointer")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, err := uc.Property.MoveItem(opCtx, interfaces.MovePropertyItemParam{
		PropertyID: pid,
		Pointer:    ptr,
		Index:      p.Index,
	}, op)
	if err != nil {
		return nil, err
	}
	return reloadSceneAfterProperty(ctx, uc, op, sid)
}
