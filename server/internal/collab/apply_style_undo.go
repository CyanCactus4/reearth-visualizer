package collab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
)

// buildUpdateStyleInverseJSON restores name/value touched by the forward update.
// If only value was changed and the previous value was nil, undo cannot be represented
// with UpdateStyle (nil skips value) — returns nil and no stack record is written.
func buildUpdateStyleInverseJSON(st *scene.Style, forward *applyUpdateStyle, touchedName, touchedVal bool) json.RawMessage {
	if st == nil || forward == nil || (!touchedName && !touchedVal) {
		return nil
	}
	inv := applyUpdateStyle{
		Kind:    "update_style",
		SceneID: forward.SceneID,
		StyleID: forward.StyleID,
	}
	if touchedName {
		n := st.Name()
		inv.Name = &n
	}
	if touchedVal {
		if ov := st.Value(); ov != nil {
			b, err := json.Marshal(map[string]any(*ov))
			if err == nil {
				inv.Value = b
			}
		}
	}
	if inv.Name == nil && len(inv.Value) == 0 {
		return nil
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// collabRunUpdateStyleFromJSON runs Style.UpdateStyle from a collab apply body (apply + undo/redo).
func collabRunUpdateStyleFromJSON(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, raw json.RawMessage) (*scene.Scene, error) {
	var p applyUpdateStyle
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		return nil, err
	}
	stid, err := id.StyleIDFrom(p.StyleID)
	if err != nil {
		return nil, err
	}
	hasVal := len(p.Value) > 0 && string(p.Value) != "null"
	var val *scene.StyleValue
	if hasVal {
		val, err = parseStyleValueRaw(p.Value)
		if err != nil {
			return nil, err
		}
	}
	if p.Name == nil && !hasVal {
		return nil, fmt.Errorf("empty_update")
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	if _, err := uc.Style.UpdateStyle(opCtx, interfaces.UpdateStyleInput{
		StyleID: stid,
		Name:    p.Name,
		Value:   val,
	}, op); err != nil {
		return nil, err
	}
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, fmt.Errorf("scene reload failed")
	}
	return scenes[0], nil
}

func buildRemoveStyleInverseJSON(styleID, sceneID string) json.RawMessage {
	inv := applyRemoveStyle{
		Kind:    "remove_style",
		SceneID: sceneID,
		StyleID: styleID,
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// buildAddStyleInverseJSON is the undo inverse for remove_style (recreate same name/value; new style id).
func buildAddStyleInverseJSON(st *scene.Style, sceneID string) json.RawMessage {
	if st == nil {
		return nil
	}
	ov := st.Value()
	if ov == nil {
		return nil
	}
	rawVal, err := json.Marshal(map[string]any(*ov))
	if err != nil {
		return nil
	}
	inv := applyAddStyle{
		Kind:    "add_style",
		SceneID: sceneID,
		Name:    st.Name(),
		Value:   json.RawMessage(rawVal),
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}
