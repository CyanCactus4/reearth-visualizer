package collab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearth/server/pkg/storytelling"
)

func nlsLayerIDListToJSON(list id.NLSLayerIDList) (json.RawMessage, error) {
	ss := make([]string, 0, len(list))
	for _, lid := range list {
		ss = append(ss, lid.String())
	}
	return json.Marshal(ss)
}

// buildUpdateStoryPageInverseJSON restores fields touched by the forward update_story_page apply.
func buildUpdateStoryPageInverseJSON(st *storytelling.Story, pg *storytelling.Page, forward *applyUpdateStoryPage) json.RawMessage {
	if st == nil || pg == nil || forward == nil {
		return nil
	}
	inv := applyUpdateStoryPage{
		Kind:    "update_story_page",
		SceneID: forward.SceneID,
		StoryID: forward.StoryID,
		PageID:  forward.PageID,
	}
	if forward.Title != nil {
		t := pg.Title()
		inv.Title = &t
	}
	if forward.Swipeable != nil {
		s := pg.Swipeable()
		inv.Swipeable = &s
	}
	if forward.Index != nil {
		idx := st.Pages().IndexOf(pg.Id())
		if idx >= 0 {
			inv.Index = &idx
		}
	}
	if forward.LayersRaw != nil {
		if raw, err := nlsLayerIDListToJSON(pg.Layers()); err == nil {
			inv.LayersRaw = raw
		}
	}
	if forward.SwipeableLayersRaw != nil {
		if raw, err := nlsLayerIDListToJSON(pg.SwipeableLayers()); err == nil {
			inv.SwipeableLayersRaw = raw
		}
	}
	if inv.Title == nil && inv.Swipeable == nil && inv.Index == nil && len(inv.LayersRaw) == 0 && len(inv.SwipeableLayersRaw) == 0 {
		return nil
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// collabRunUpdateStoryPageFromJSON runs StoryTelling.UpdatePage from a collab apply body (undo/redo).
func collabRunUpdateStoryPageFromJSON(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, raw json.RawMessage) (*scene.Scene, error) {
	var p applyUpdateStoryPage
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		return nil, err
	}
	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		return nil, err
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		return nil, err
	}
	if p.Title == nil && p.Swipeable == nil && p.Index == nil && p.LayersRaw == nil && p.SwipeableLayersRaw == nil {
		return nil, fmt.Errorf("empty_update")
	}
	param, err := buildUpdatePageParamFromApply(&p, sid, storyID, pageID)
	if err != nil {
		return nil, err
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	if _, _, err := uc.StoryTelling.UpdatePage(opCtx, param, op); err != nil {
		return nil, err
	}
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, fmt.Errorf("scene reload failed")
	}
	return scenes[0], nil
}
