package collab

import (
	"context"
	"encoding/json"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/pkg/id"
)

type applyMergePropertyJSON struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	PropertyID   string          `json:"propertyId"`
	Patch        json.RawMessage `json:"patch"`
	DocClock     *int64          `json:"docClock,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
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

	// Same mutex as `update_property_value` (HLC / int field clock) so batch merge + single-field applies don't interleave.
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

	var patchTop map[string]any
	if err := json.Unmarshal(p.Patch, &patchTop); err != nil || patchTop == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "patch must be a JSON object"}})
		return nil
	}
	if len(patchTop) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "empty patch"}})
		return nil
	}

	origLeaves, err := flattenPropertyValueLeaves(prop)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	origGen, err := leavesToGeneric(origLeaves)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	merged := jsonMergePatchObject(origGen, patchTop)
	for k := range origGen {
		if _, ok := merged[k]; !ok {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
				"code":    "invalid_payload",
				"message": "merge patch cannot remove existing property fields (null keys)",
			}})
			return nil
		}
	}

	var updatedKeys []string
	for k, newLeaf := range merged {
		oldLeaf, had := origGen[k]
		if !had {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
				"code":    "invalid_payload",
				"message": "unknown field key in patch: " + k,
			}})
			return nil
		}
		nm, ok := newLeaf.(map[string]any)
		if !ok {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "patch leaf must be object with type/value"}})
			return nil
		}
		if leafMapsEqualJSON(toStringAnyMap(oldLeaf), nm) {
			continue
		}
		sg, itemID, fieldID, ok := parseFlatPropertyFieldKey(k)
		if !ok {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid field key"}})
			return nil
		}
		tStr, _ := nm["type"].(string)
		if tStr == "" {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "missing type"}})
			return nil
		}
		vt := gqlmodel.ValueType(tStr)
		var valIface interface{}
		if rawVal, ok := nm["value"]; ok && rawVal != nil {
			vb, err := json.Marshal(rawVal)
			if err != nil {
				from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
				return nil
			}
			if err := json.Unmarshal(vb, &valIface); err != nil {
				from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
				return nil
			}
		}
		val := gqlmodel.FromPropertyValueAndType(valIface, vt)
		if val == nil && valIface != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid value for type " + tStr}})
			return nil
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
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "invalid pointer"}})
			return nil
		}
		if _, err := runPropertyValueUpdate(ctx, uc, op, pid, ptr, val, sid); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
			return nil
		}
		if hub != nil {
			hub.BumpPropertyFieldClock(sid.String(), p.PropertyID, sg, itemID, fieldID)
		}
		updatedKeys = append(updatedKeys, k)
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": errSceneReloadFailed.Error()}})
		return nil
	}
	sc := scenes[0]

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
	return nil
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
