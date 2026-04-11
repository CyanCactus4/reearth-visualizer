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
)

type applyUpdatePropertyValue struct {
	Kind          string          `json:"kind"`
	SceneID       string          `json:"sceneId"`
	PropertyID    string          `json:"propertyId"`
	SchemaGroupID *string         `json:"schemaGroupId,omitempty"`
	ItemID        *string         `json:"itemId,omitempty"`
	FieldID       string          `json:"fieldId"`
	Type          string          `json:"type"`
	Value         json.RawMessage `json:"value,omitempty"`
	BaseSceneRev  *int64          `json:"baseSceneRev,omitempty"`
}

func sceneMustNotBeLockedByPeer(ctx context.Context, hub *Hub, from *Conn, sid id.SceneID) bool {
	holder, active, err := hub.LockHolder(ctx, from.projectID, "scene", sid.String())
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_lookup", "message": err.Error()}})
		return false
	}
	if !active {
		return true
	}
	if holder == from.userID {
		return true
	}
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "object_locked", "message": "scene locked by " + holder}})
	return false
}

func propertyBelongsToScene(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID, from *Conn) bool {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Property.Fetch(opCtx, []id.PropertyID{pid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "property not found"}})
		return false
	}
	if list[0].Scene() != sid {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "property does not belong to this scene"}})
		return false
	}
	return true
}

func applyUpdatePropertyValueOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdatePropertyValue
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
	if !propertyBelongsToScene(ctx, uc, op, sid, pid, from) {
		return nil
	}
	vt := gqlmodel.ValueType(p.Type)
	var val *property.Value
	hasValue := len(p.Value) > 0 && string(p.Value) != "null"
	if hasValue {
		var valIface interface{}
		if err := json.Unmarshal(p.Value, &valIface); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
			return nil
		}
		val = gqlmodel.FromPropertyValueAndType(valIface, vt)
		if val == nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid value"}})
			return nil
		}
	}

	var schemaGID *gqlmodel.ID
	if p.SchemaGroupID != nil && *p.SchemaGroupID != "" {
		g := gqlmodel.ID(*p.SchemaGroupID)
		schemaGID = &g
	}
	var itemGID *gqlmodel.ID
	if p.ItemID != nil && *p.ItemID != "" {
		g := gqlmodel.ID(*p.ItemID)
		itemGID = &g
	}
	fid := gqlmodel.ID(p.FieldID)
	ptr := gqlmodel.FromPointer(
		gqlmodel.ToStringIDRef[id.PropertySchemaGroup](schemaGID),
		itemGID,
		gqlmodel.ToStringIDRef[id.PropertyField](&fid),
	)
	if ptr == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid property pointer"}})
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, _, err2 := uc.Property.UpdateValue(opCtx, interfaces.UpdatePropertyValueParam{
		PropertyID: pid,
		Pointer:    ptr,
		Value:      val,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "update_property_value", map[string]any{
		"sceneId":    p.SceneID,
		"propertyId": p.PropertyID,
		"fieldId":    p.FieldID,
	}, sc)
	return nil
}
