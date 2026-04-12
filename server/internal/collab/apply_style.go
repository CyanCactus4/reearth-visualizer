package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
)

type applyAddStyle struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	Name         string          `json:"name"`
	Value        json.RawMessage `json:"value"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyUpdateStyle struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	StyleID      string          `json:"styleId"`
	Name         *string         `json:"name,omitempty"`
	Value        json.RawMessage `json:"value,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyRemoveStyle struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	StyleID      string `json:"styleId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func parseStyleValueRaw(raw json.RawMessage) (*scene.StyleValue, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	sv := scene.StyleValue(m)
	return &sv, nil
}

func styleMustNotBeLockedByPeer(ctx context.Context, hub *Hub, from *Conn, sid id.StyleID) bool {
	if hub == nil {
		return true
	}
	holder, active, err := hub.LockHolder(ctx, from.projectID, "style", sid.String())
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_lookup", "message": err.Error()}})
		return false
	}
	if !active {
		return true
	}
	if LockHeldBySameTab(holder, from.userID, from.clientID) {
		return true
	}
	hu, _ := ParseLockHolderWire(holder)
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "object_locked", "message": "style locked by " + hu}})
	return false
}

func fetchStyleForCollabApply(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, stid id.StyleID, from *Conn) *scene.Style {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.Style.Fetch(opCtx, id.StyleIDList{stid}, op)
	if err != nil || list == nil || len(*list) == 0 || (*list)[0] == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "style not found"}})
		return nil
	}
	st := (*list)[0]
	if st.Scene() != sid {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "style does not belong to this scene"}})
		return nil
	}
	return st
}

func applyAddStyleOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddStyle
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
	val, err := parseStyleValueRaw(p.Value)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	if val == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "value required"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	st, err2 := uc.Style.AddStyle(opCtx, interfaces.AddStyleInput{
		SceneID: sid,
		Name:    p.Name,
		Value:   val,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	sidStr := ""
	if st != nil {
		sidStr = st.ID().String()
	}
	broadcastApplied(ctx, hub, from, "add_style", map[string]any{
		"sceneId": p.SceneID,
		"styleId": sidStr,
	}, sc)
	if hub != nil && hub.opStack != nil && st != nil {
		invJSON := buildRemoveStyleInverseJSON(st.ID().String(), p.SceneID)
		if len(invJSON) > 0 {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_style",
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

func applyUpdateStyleOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateStyle
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
	stid, err := id.StyleIDFrom(p.StyleID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_style", "message": err.Error()}})
		return nil
	}
	st := fetchStyleForCollabApply(ctx, uc, op, sid, stid, from)
	if st == nil {
		return nil
	}
	if !styleMustNotBeLockedByPeer(ctx, hub, from, stid) {
		return nil
	}
	hasVal := len(p.Value) > 0 && string(p.Value) != "null"
	if p.Name == nil && !hasVal {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "no style fields to update"}})
		return nil
	}
	var val *scene.StyleValue
	if hasVal {
		var errV error
		val, errV = parseStyleValueRaw(p.Value)
		if errV != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": errV.Error()}})
			return nil
		}
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		invJSON = buildUpdateStyleInverseJSON(st, &p, p.Name != nil, hasVal)
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.Style.UpdateStyle(opCtx, interfaces.UpdateStyleInput{
		StyleID: stid,
		Name:    p.Name,
		Value:   val,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "update_style", map[string]any{
		"sceneId": p.SceneID,
		"styleId": p.StyleID,
	}, sc)

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "update_style",
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

func applyRemoveStyleOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveStyle
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
	stid, err := id.StyleIDFrom(p.StyleID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_style", "message": err.Error()}})
		return nil
	}
	st := fetchStyleForCollabApply(ctx, uc, op, sid, stid, from)
	if st == nil {
		return nil
	}
	if !styleMustNotBeLockedByPeer(ctx, hub, from, stid) {
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		invJSON = buildAddStyleInverseJSON(st, p.SceneID)
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.Style.RemoveStyle(opCtx, stid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_style", map[string]any{
		"sceneId": p.SceneID,
		"styleId": p.StyleID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "remove_style",
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
