package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type applyMergePropertyJSON struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	PropertyID   string          `json:"propertyId"`
	Patch        json.RawMessage `json:"patch"`
	DocClock     *int64          `json:"docClock,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

// mergePropertyJSONErr carries collab wire error codes for apply vs undo paths.
type mergePropertyJSONErr struct {
	code string
	msg  string
}

func (e *mergePropertyJSONErr) Error() string {
	return e.msg
}

func loadPropertyForMergeJSON(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, pid id.PropertyID) (*property.Property, error) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Property.Fetch(opCtx, []id.PropertyID{pid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		return nil, &mergePropertyJSONErr{"apply_failed", "property not found"}
	}
	if list[0].Scene() != sid {
		return nil, &mergePropertyJSONErr{"scene_mismatch", "property does not belong to this scene"}
	}
	return list[0], nil
}

func sceneMustNotBeLockedForMergeUndo(ctx context.Context, hub *Hub, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID) error {
	if hub == nil {
		return nil
	}
	u := adapter.User(ctx)
	if u == nil {
		return nil
	}
	uid := u.ID().String()
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return &mergePropertyJSONErr{"internal", "scene fetch failed"}
	}
	prj := scenes[0].Project().String()
	holder, active, err := hub.LockHolder(ctx, prj, "scene", sid.String())
	if err != nil {
		return err
	}
	if active && HTTPLockBlocksUser(holder, uid) {
		hu, _ := ParseLockHolderWire(holder)
		return &mergePropertyJSONErr{"object_locked", "scene locked by " + hu}
	}
	return nil
}

// applyMergePatchToProperty applies merged flat leaves, bumps per-field clocks when hub is set, and returns
// inverse patch leaves (old type/value) for keys that changed — used for undo stack.
func applyMergePatchToProperty(
	ctx context.Context,
	hub *Hub,
	uc *interfaces.Container,
	op *usecase.Operator,
	sid id.SceneID,
	pid id.PropertyID,
	p *applyMergePropertyJSON,
	prop *property.Property,
) (*scene.Scene, []string, map[string]any, error) {
	var patchTop map[string]any
	if err := json.Unmarshal(p.Patch, &patchTop); err != nil || patchTop == nil {
		return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "patch must be a JSON object"}
	}
	if len(patchTop) == 0 {
		return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "empty patch"}
	}

	origLeaves, err := flattenPropertyValueLeaves(prop)
	if err != nil {
		return nil, nil, nil, &mergePropertyJSONErr{"apply_failed", err.Error()}
	}
	origGen, err := leavesToGeneric(origLeaves)
	if err != nil {
		return nil, nil, nil, &mergePropertyJSONErr{"apply_failed", err.Error()}
	}
	merged := jsonMergePatchObject(origGen, patchTop)
	for k := range origGen {
		if _, ok := merged[k]; !ok {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "merge patch cannot remove existing property fields (null keys)"}
		}
	}

	inversePatch := make(map[string]any)
	var updatedKeys []string
	for k, newLeaf := range merged {
		oldLeaf, had := origGen[k]
		if !had {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "unknown field key in patch: " + k}
		}
		nm, ok := newLeaf.(map[string]any)
		if !ok {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "patch leaf must be object with type/value"}
		}
		if leafMapsEqualJSON(toStringAnyMap(oldLeaf), nm) {
			continue
		}
		sg, itemID, fieldID, ok := parseFlatPropertyFieldKey(k)
		if !ok {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "invalid field key"}
		}
		tStr, _ := nm["type"].(string)
		if tStr == "" {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "missing type"}
		}
		vt := gqlmodel.ValueType(tStr)
		var valIface interface{}
		if rawVal, ok := nm["value"]; ok && rawVal != nil {
			vb, err := json.Marshal(rawVal)
			if err != nil {
				return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", err.Error()}
			}
			if err := json.Unmarshal(vb, &valIface); err != nil {
				return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", err.Error()}
			}
		}
		val := gqlmodel.FromPropertyValueAndType(valIface, vt)
		if val == nil && valIface != nil {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "invalid value for type " + tStr}
		}
		sgID := id.PropertySchemaGroupID(sg)
		var itemGql *gqlmodel.ID
		if itemID != "" {
			g := gqlmodel.ID(itemID)
			itemGql = &g
		}
		fid := gqlmodel.ID(fieldID)
		ptr := gqlmodel.FromPointer(&sgID, itemGql, gqlmodel.ToStringIDRef[id.PropertyField](&fid))
		if ptr == nil {
			return nil, nil, nil, &mergePropertyJSONErr{"invalid_payload", "invalid pointer"}
		}
		if _, err := runPropertyValueUpdate(ctx, uc, op, pid, ptr, val, sid); err != nil {
			return nil, nil, nil, &mergePropertyJSONErr{"apply_failed", err.Error()}
		}
		if hub != nil {
			hub.BumpPropertyFieldClock(sid.String(), p.PropertyID, sg, itemID, fieldID)
		}
		updatedKeys = append(updatedKeys, k)
		inversePatch[k] = cloneJSONMapStringAny(toStringAnyMap(oldLeaf))
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, nil, nil, &mergePropertyJSONErr{"apply_failed", errSceneReloadFailed.Error()}
	}
	sc := scenes[0]
	return sc, updatedKeys, inversePatch, nil
}

func enqueueMergeErr(from *Conn, e *mergePropertyJSONErr) {
	if from == nil || e == nil {
		return
	}
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": e.code, "message": e.msg}})
}

func applyMergePropertyJSONOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyMergePropertyJSON
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

	if hub != nil {
		hub.propertyCollabApplyMu.Lock()
		defer hub.propertyCollabApplyMu.Unlock()
	}

	docClockUsed := p.DocClock != nil && hub != nil
	if !docClockUsed {
		if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
			return nil
		}
	} else if hub != nil {
		srv := hub.PropertyDocClock(sid.String(), p.PropertyID)
		if srv != *p.DocClock {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
				"code":    "stale_property_doc",
				"message": "property doc clock mismatch; refetch scene and retry merge",
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

	sc, updatedKeys, inversePatch, err2 := applyMergePatchToProperty(ctx, hub, uc, op, sid, pid, &p, prop)
	if err2 != nil {
		var me *mergePropertyJSONErr
		if errors.As(err2, &me) {
			enqueueMergeErr(from, me)
		} else {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		}
		return nil
	}

	var nextDocClock int64
	if hub != nil {
		nextDocClock = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	extra := map[string]any{
		"sceneId":    p.SceneID,
		"propertyId": p.PropertyID,
		"mergedKeys": updatedKeys,
	}
	if hub != nil {
		extra["propertyDocClock"] = nextDocClock
	}
	broadcastApplied(ctx, hub, from, "merge_property_json", extra, sc)

	if hub != nil && hub.opStack != nil && len(updatedKeys) > 0 && len(inversePatch) > 0 {
		pb, err := json.Marshal(inversePatch)
		if err == nil {
			inv := applyMergePropertyJSON{
				Kind:       "merge_property_json",
				SceneID:    p.SceneID,
				PropertyID: p.PropertyID,
				Patch:      pb,
			}
			invJSON, mErr := json.Marshal(&inv)
			if mErr == nil {
				rec := UndoableOpRecord{
					ProjectID: from.projectID,
					SceneID:   sid.String(),
					UserID:    actorUserID(from),
					Kind:      "merge_property_json",
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
	}
	return nil
}

// mergePropertyJSONForUndoRedo replays merge_property_json (used by POST /api/collab/undo|redo).
func mergePropertyJSONForUndoRedo(ctx context.Context, hub *Hub, uc *interfaces.Container, op *usecase.Operator, raw json.RawMessage) (*scene.Scene, error) {
	var p applyMergePropertyJSON
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		return nil, err
	}
	if op == nil || !op.IsWritableScene(sid) {
		return nil, fmt.Errorf("write not allowed")
	}
	if hub != nil {
		hub.propertyCollabApplyMu.Lock()
		defer hub.propertyCollabApplyMu.Unlock()
	}
	docClockUsed := p.DocClock != nil && hub != nil
	if !docClockUsed {
		if err := assertSceneRevIfPresentErr(ctx, uc, op, sid, raw); err != nil {
			if errors.Is(err, errStaleSceneRev) {
				return nil, fmt.Errorf("stale_state")
			}
			return nil, err
		}
	} else if hub != nil {
		srv := hub.PropertyDocClock(sid.String(), p.PropertyID)
		if srv != *p.DocClock {
			return nil, fmt.Errorf("stale_property_doc")
		}
	}
	if err := sceneMustNotBeLockedForMergeUndo(ctx, hub, uc, op, sid); err != nil {
		var me *mergePropertyJSONErr
		if errors.As(err, &me) {
			return nil, fmt.Errorf("%s: %s", me.code, me.msg)
		}
		return nil, err
	}
	pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
	if err != nil {
		return nil, err
	}
	prop, err := loadPropertyForMergeJSON(ctx, uc, op, sid, pid)
	if err != nil {
		return nil, err
	}
	sc, _, _, err2 := applyMergePatchToProperty(ctx, hub, uc, op, sid, pid, &p, prop)
	if err2 != nil {
		var me *mergePropertyJSONErr
		if errors.As(err2, &me) {
			return nil, fmt.Errorf("%s: %s", me.code, me.msg)
		}
		return nil, err2
	}
	if hub != nil {
		_ = hub.BumpPropertyDocClock(sid.String(), p.PropertyID)
	}
	return sc, nil
}

func leavesToGeneric(m map[string]map[string]any) (map[string]any, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func toStringAnyMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}
