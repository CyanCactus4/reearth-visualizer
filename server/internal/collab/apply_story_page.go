package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/storytelling"
	"github.com/reearth/reearthx/log"
)

type applyCreateStoryPage struct {
	Kind            string   `json:"kind"`
	SceneID         string   `json:"sceneId"`
	StoryID         string   `json:"storyId"`
	Title           *string  `json:"title,omitempty"`
	Swipeable       *bool    `json:"swipeable,omitempty"`
	Layers          []string `json:"layers,omitempty"`
	SwipeableLayers []string `json:"swipeableLayers,omitempty"`
	Index           *int     `json:"index,omitempty"`
	BaseSceneRev    *int64   `json:"baseSceneRev,omitempty"`
}

type applyRemoveStoryPage struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	StoryID      string `json:"storyId"`
	PageID       string `json:"pageId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyMoveStoryPage struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	StoryID      string `json:"storyId"`
	PageID       string `json:"pageId"`
	Index        int    `json:"index"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyDuplicateStoryPage struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	StoryID      string `json:"storyId"`
	PageID       string `json:"pageId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyUpdateStoryPage struct {
	Kind               string          `json:"kind"`
	SceneID            string          `json:"sceneId"`
	StoryID            string          `json:"storyId"`
	PageID             string          `json:"pageId"`
	Title              *string         `json:"title,omitempty"`
	Swipeable          *bool           `json:"swipeable,omitempty"`
	Index              *int            `json:"index,omitempty"`
	BaseSceneRev       *int64          `json:"baseSceneRev,omitempty"`
	LayersRaw          json.RawMessage `json:"layers,omitempty"`
	SwipeableLayersRaw json.RawMessage `json:"swipeableLayers,omitempty"`
}

func parseNLSLayerIDList(ss []string) ([]id.NLSLayerID, error) {
	out := make([]id.NLSLayerID, 0, len(ss))
	for _, s := range ss {
		lid, err := id.NLSLayerIDFrom(s)
		if err != nil {
			return nil, err
		}
		out = append(out, lid)
	}
	return out, nil
}

func fetchStoryPageForCollabApply(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, storyID id.StoryID, pageID id.PageID, from *Conn) (*storytelling.Story, *storytelling.Page) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	sl, err := uc.StoryTelling.Fetch(opCtx, id.StoryIDList{storyID}, op)
	if err != nil || sl == nil || len(*sl) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "story not found"}})
		return nil, nil
	}
	var st *storytelling.Story
	for _, s := range *sl {
		if s != nil && s.Id() == storyID {
			st = s
			break
		}
	}
	if st == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "story not found"}})
		return nil, nil
	}
	pg := st.Pages().Page(pageID)
	if pg == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "page not found"}})
		return nil, nil
	}
	return st, pg
}

func buildUpdatePageParamFromApply(p *applyUpdateStoryPage, sid id.SceneID, storyID id.StoryID, pageID id.PageID) (interfaces.UpdatePageParam, error) {
	param := interfaces.UpdatePageParam{
		SceneID:   sid,
		StoryID:   storyID,
		PageID:    pageID,
		Title:     p.Title,
		Index:     p.Index,
		Swipeable: p.Swipeable,
	}
	if p.LayersRaw != nil {
		var ss []string
		if err := json.Unmarshal(p.LayersRaw, &ss); err != nil {
			return param, err
		}
		layers, errL := parseNLSLayerIDList(ss)
		if errL != nil {
			return param, errL
		}
		param.Layers = &layers
	}
	if p.SwipeableLayersRaw != nil {
		var ss []string
		if err := json.Unmarshal(p.SwipeableLayersRaw, &ss); err != nil {
			return param, err
		}
		sl, errS := parseNLSLayerIDList(ss)
		if errS != nil {
			return param, errS
		}
		param.SwipeableLayers = &sl
	}
	return param, nil
}

func applyCreateStoryPageOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyCreateStoryPage
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
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	param := interfaces.CreatePageParam{
		SceneID: sid,
		StoryID: storyID,
		Title:   p.Title,
		Index:   p.Index,
	}
	if p.Swipeable != nil {
		param.Swipeable = p.Swipeable
	}
	if len(p.Layers) > 0 {
		layers, errL := parseNLSLayerIDList(p.Layers)
		if errL != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": errL.Error()}})
			return nil
		}
		param.Layers = &layers
	}
	if len(p.SwipeableLayers) > 0 {
		sl, errS := parseNLSLayerIDList(p.SwipeableLayers)
		if errS != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": errS.Error()}})
			return nil
		}
		param.SwipeableLayers = &sl
	}
	_, page, err2 := uc.StoryTelling.CreatePage(opCtx, param, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	extra := map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
	}
	if page != nil {
		extra["pageId"] = page.Id().String()
	}
	broadcastApplied(ctx, hub, from, "create_story_page", extra, sc)
	return nil
}

func applyRemoveStoryPageOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveStoryPage
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
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_page", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, err2 := uc.StoryTelling.RemovePage(opCtx, interfaces.RemovePageParam{
		SceneID: sid,
		StoryID: storyID,
		PageID:  pageID,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	broadcastApplied(ctx, hub, from, "remove_story_page", map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
	}, sc)
	return nil
}

func applyMoveStoryPageOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyMoveStoryPage
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
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_page", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	fromIdx := -1
	if sl, e := uc.StoryTelling.Fetch(opCtx, id.StoryIDList{storyID}, op); e == nil && sl != nil && len(*sl) > 0 {
		for _, st := range *sl {
			if st == nil || st.Id() != storyID {
				continue
			}
			if pl := st.Pages(); pl != nil {
				fromIdx = pl.IndexOf(pageID)
			}
			break
		}
	}

	_, _, _, err2 := uc.StoryTelling.MovePage(opCtx, interfaces.MovePageParam{
		StoryID: storyID,
		PageID:  pageID,
		Index:   p.Index,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	broadcastApplied(ctx, hub, from, "move_story_page", map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
	}, sc)

	if hub != nil && hub.opStack != nil && fromIdx >= 0 {
		inv := applyMoveStoryPage{
			Kind:    "move_story_page",
			SceneID: p.SceneID,
			StoryID: p.StoryID,
			PageID:  p.PageID,
			Index:   fromIdx,
		}
		binv, errI := json.Marshal(inv)
		if errI == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "move_story_page",
				Forward:   d,
				Inverse:   json.RawMessage(binv),
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

func applyUpdateStoryPageOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateStoryPage
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
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_page", "message": err.Error()}})
		return nil
	}

	if p.Title == nil && p.Swipeable == nil && p.Index == nil && p.LayersRaw == nil && p.SwipeableLayersRaw == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "no story page fields to update"}})
		return nil
	}

	st, pg := fetchStoryPageForCollabApply(ctx, uc, op, storyID, pageID, from)
	if st == nil || pg == nil {
		return nil
	}
	param, errP := buildUpdatePageParamFromApply(&p, sid, storyID, pageID)
	if errP != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": errP.Error()}})
		return nil
	}

	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		invJSON = buildUpdateStoryPageInverseJSON(st, pg, &p)
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, err2 := uc.StoryTelling.UpdatePage(opCtx, param, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	broadcastApplied(ctx, hub, from, "update_story_page", map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
	}, sc)

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "update_story_page",
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

func applyDuplicateStoryPageOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyDuplicateStoryPage
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
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_page", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, dupPage, err2 := uc.StoryTelling.DuplicatePage(opCtx, interfaces.DuplicatePageParam{
		SceneID: sid,
		StoryID: storyID,
		PageID:  pageID,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	extra := map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
	}
	if dupPage != nil {
		extra["duplicatedPageId"] = dupPage.Id().String()
	}
	broadcastApplied(ctx, hub, from, "duplicate_story_page", extra, sc)
	return nil
}
