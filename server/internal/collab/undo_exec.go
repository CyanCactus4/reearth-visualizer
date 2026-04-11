package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/adapter/gql/gqlmodel"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
)

// ExecuteCollabUndoJSON runs one undo/redo payload (inverse or forward JSON) through the same interactors as collab apply.
func ExecuteCollabUndoJSON(ctx context.Context, raw json.RawMessage, operator *usecase.Operator) (*scene.Scene, error) {
	var env struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	uc := adapter.Usecases(ctx)
	if uc == nil {
		return nil, errors.New("usecases unavailable")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	switch env.Kind {
	case "update_widget":
		var p applyUpdateWidget
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		sid, err := id.SceneIDFrom(p.SceneID)
		if err != nil {
			return nil, err
		}
		wid, err := id.WidgetIDFrom(p.WidgetID)
		if err != nil {
			return nil, err
		}
		align, err := parseAlignSystem(p.AlignSystem)
		if err != nil {
			return nil, err
		}
		var loc *scene.WidgetLocation
		if p.Location != nil {
			loc = &scene.WidgetLocation{
				Zone:    scene.WidgetZoneType(p.Location.Zone),
				Section: scene.WidgetSectionType(p.Location.Section),
				Area:    scene.WidgetAreaType(p.Location.Area),
			}
			if !widgetLocationValid(*loc) {
				return nil, fmt.Errorf("invalid widget location")
			}
		}
		if p.Enabled == nil && p.Extended == nil && loc == nil && p.Index == nil {
			return nil, fmt.Errorf("empty update_widget payload")
		}
		sc, _, err := uc.Scene.UpdateWidget(opCtx, interfaces.UpdateWidgetParam{
			Type:     align,
			SceneID:  sid,
			WidgetID: wid,
			Enabled:  p.Enabled,
			Extended: p.Extended,
			Location: loc,
			Index:    p.Index,
		}, operator)
		return sc, err
	case "move_story_block":
		var p applyMoveStoryBlock
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
		blockID, err := id.BlockIDFrom(p.BlockID)
		if err != nil {
			return nil, err
		}
		_, _, _, _, errM := uc.StoryTelling.MoveBlock(opCtx, interfaces.MoveBlockParam{
			StoryID: storyID,
			PageID:  pageID,
			BlockID: blockID,
			Index:   p.Index,
		}, operator)
		if errM != nil {
			return nil, errM
		}
		scenes, err2 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, operator)
		if err2 != nil || len(scenes) == 0 {
			return nil, fmt.Errorf("scene reload failed")
		}
		return scenes[0], nil
	case "update_property_value":
		p, ptr, val, err := decodeApplyUpdatePropertyValue(raw)
		if err != nil {
			return nil, err
		}
		sid, err := id.SceneIDFrom(p.SceneID)
		if err != nil {
			return nil, err
		}
		if operator == nil || !operator.IsWritableScene(sid) {
			return nil, errors.New("write not allowed")
		}
		pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
		if err != nil {
			return nil, err
		}
		return runPropertyValueUpdate(ctx, uc, operator, pid, ptr, val, sid)
	case "update_style":
		var p applyUpdateStyle
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		sid, err := id.SceneIDFrom(p.SceneID)
		if err != nil {
			return nil, err
		}
		if operator == nil || !operator.IsWritableScene(sid) {
			return nil, errors.New("write not allowed")
		}
		return collabRunUpdateStyleFromJSON(ctx, uc, operator, raw)
	case "update_story_page":
		var p applyUpdateStoryPage
		if err := json.Unmarshal(raw, &p); err != nil {
			return nil, err
		}
		sid, err := id.SceneIDFrom(p.SceneID)
		if err != nil {
			return nil, err
		}
		if operator == nil || !operator.IsWritableScene(sid) {
			return nil, errors.New("write not allowed")
		}
		return collabRunUpdateStoryPageFromJSON(ctx, uc, operator, raw)
	default:
		return nil, fmt.Errorf("unsupported undo kind %q", env.Kind)
	}
}
