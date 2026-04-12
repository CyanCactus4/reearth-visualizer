package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/property"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
)

type fieldHLCWire struct {
	Wall    int64  `json:"wall"`
	Logical uint32 `json:"logical"`
	Node    string `json:"node"`
}

func (w *fieldHLCWire) toHLC() HLC {
	if w == nil {
		return ZeroHLC
	}
	return HLC{Physical: w.Wall, Logical: w.Logical, NodeID: w.Node}
}

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
	// FieldClock optional per-field LWW clock (see Hub.PropertyFieldClock). When set, baseSceneRev is not required.
	FieldClock *int64 `json:"fieldClock,omitempty"`
	// FieldHLC optional Hybrid Logical Clock (LWW-register CRDT). When set with hub, baseSceneRev is not required; FieldClock is ignored.
	FieldHLC *fieldHLCWire `json:"fieldHlc,omitempty"`
}

func sceneMustNotBeLockedByPeer(ctx context.Context, hub *Hub, from *Conn, sid id.SceneID) bool {
	if hub == nil {
		return true
	}
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

func fetchPropertyForCollabApply(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID, from *Conn) *property.Property {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Property.Fetch(opCtx, []id.PropertyID{pid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "property not found"}})
		return nil
	}
	if list[0].Scene() != sid {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "property does not belong to this scene"}})
		return nil
	}
	return list[0]
}

// decodeApplyUpdatePropertyValue unmarshals the apply body and builds pointer + value for Property.UpdateValue.
func decodeApplyUpdatePropertyValue(d json.RawMessage) (p applyUpdatePropertyValue, ptr *property.Pointer, val *property.Value, err error) {
	if err = json.Unmarshal(d, &p); err != nil {
		return
	}
	vt := gqlmodel.ValueType(p.Type)
	hasValue := len(p.Value) > 0 && string(p.Value) != "null"
	if hasValue {
		var valIface interface{}
		if err = json.Unmarshal(p.Value, &valIface); err != nil {
			return
		}
		val = gqlmodel.FromPropertyValueAndType(valIface, vt)
		if val == nil {
			err = errInvalidPropertyApplyValue
			return
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
	ptr = gqlmodel.FromPointer(
		gqlmodel.ToStringIDRef[id.PropertySchemaGroup](schemaGID),
		itemGID,
		gqlmodel.ToStringIDRef[id.PropertyField](&fid),
	)
	if ptr == nil {
		err = errInvalidPropertyApplyPointer
	}
	return
}

var (
	errInvalidPropertyApplyValue   = errInvalidPropertyApply("invalid value")
	errInvalidPropertyApplyPointer = errInvalidPropertyApply("invalid property pointer")
)

type errInvalidPropertyApply string

func (e errInvalidPropertyApply) Error() string { return string(e) }

func runPropertyValueUpdate(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, pid id.PropertyID, ptr *property.Pointer, val *property.Value, sid id.SceneID) (*scene.Scene, error) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, _, _, err := uc.Property.UpdateValue(opCtx, interfaces.UpdatePropertyValueParam{
		PropertyID: pid,
		Pointer:    ptr,
		Value:      val,
	}, op)
	if err != nil {
		return nil, err
	}
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, errSceneReloadFailed
	}
	return scenes[0], nil
}

var errSceneReloadFailed = errInvalidPropertyApply("scene reload failed")

func applyUpdatePropertyValueOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	p, ptr, val, err := decodeApplyUpdatePropertyValue(d)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
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
	fieldHlcUsed := p.FieldHLC != nil && hub != nil
	fieldClockUsed := p.FieldClock != nil && hub != nil && !fieldHlcUsed

	if !fieldHlcUsed && !fieldClockUsed {
		if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
			return nil
		}
	}
	if fieldHlcUsed || fieldClockUsed {
		hub.propertyCollabApplyMu.Lock()
		defer hub.propertyCollabApplyMu.Unlock()
	}
	if fieldHlcUsed {
		inc := p.FieldHLC.toHLC()
		inc.NodeID = normalizeNodeID(inc.NodeID)
		if !inc.IsValidClientHLC() {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid fieldHlc (wall/logical/node required)"}})
			return nil
		}
		sg := ""
		if p.SchemaGroupID != nil {
			sg = *p.SchemaGroupID
		}
		it := ""
		if p.ItemID != nil {
			it = *p.ItemID
		}
		if !inc.After(hub.PropertyFieldHLC(sid.String(), p.PropertyID, sg, it, p.FieldID)) {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
				"code":    "stale_property_field",
				"message": "property field HLC not newer than server; refresh from last applied",
			}})
			return nil
		}
	} else if fieldClockUsed {
		sg := ""
		if p.SchemaGroupID != nil {
			sg = *p.SchemaGroupID
		}
		it := ""
		if p.ItemID != nil {
			it = *p.ItemID
		}
		srv := hub.PropertyFieldClock(sid.String(), p.PropertyID, sg, it, p.FieldID)
		if srv > *p.FieldClock {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
				"code":    "stale_property_field",
				"message": "property field clock behind server; refresh from last applied",
			}})
			return nil
		}
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
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		invJSON = buildUpdatePropertyValueInverseJSON(prop, &p, ptr)
	}
	sc, err2 := runPropertyValueUpdate(ctx, uc, op, pid, ptr, val, sid)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	extra := map[string]any{
		"sceneId":    p.SceneID,
		"propertyId": p.PropertyID,
		"fieldId":    p.FieldID,
	}
	if p.SchemaGroupID != nil && *p.SchemaGroupID != "" {
		extra["schemaGroupId"] = *p.SchemaGroupID
	}
	if p.ItemID != nil && *p.ItemID != "" {
		extra["itemId"] = *p.ItemID
	}
	if fieldClockUsed && hub != nil {
		sg := ""
		if p.SchemaGroupID != nil {
			sg = *p.SchemaGroupID
		}
		it := ""
		if p.ItemID != nil {
			it = *p.ItemID
		}
		extra["propertyFieldClock"] = hub.BumpPropertyFieldClock(sid.String(), p.PropertyID, sg, it, p.FieldID)
	}
	if fieldHlcUsed && hub != nil {
		sg := ""
		if p.SchemaGroupID != nil {
			sg = *p.SchemaGroupID
		}
		it := ""
		if p.ItemID != nil {
			it = *p.ItemID
		}
		committed := hub.advancePropertyFieldHLC(sid.String(), p.PropertyID, sg, it, p.FieldID, p.FieldHLC.toHLC(), time.Now().UnixMilli())
		extra["propertyFieldHlc"] = map[string]any{
			"wall":     committed.Physical,
			"logical":  committed.Logical,
			"node":     committed.NodeID,
		}
		extra["propertyFieldClock"] = hub.BumpPropertyFieldClock(sid.String(), p.PropertyID, sg, it, p.FieldID)
	}
	if hub != nil {
		extra["propertyDocClock"] = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	broadcastApplied(ctx, hub, from, "update_property_value", extra, sc)

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "update_property_value",
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
