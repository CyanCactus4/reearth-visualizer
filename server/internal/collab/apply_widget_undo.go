package collab

import (
	"encoding/json"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
)

func buildRemoveWidgetInverseJSON(sidStr, alignStr, widStr string) json.RawMessage {
	inv := applyRemoveWidget{
		Kind:        "remove_widget",
		SceneID:     sidStr,
		AlignSystem: alignStr,
		WidgetID:    widStr,
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// buildAddWidgetInverseJSON is the undo inverse for remove_widget (re-add same plugin/extension).
func buildAddWidgetInverseJSON(sc *scene.Scene, sidStr, alignStr string, wid id.WidgetID) json.RawMessage {
	if sc == nil {
		return nil
	}
	w := sc.Widgets().Widget(wid)
	if w == nil {
		return nil
	}
	inv := applyAddWidget{
		Kind:        "add_widget",
		SceneID:     sidStr,
		AlignSystem: alignStr,
		PluginID:    w.Plugin().String(),
		ExtensionID: string(w.Extension()),
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

func buildUpdateWidgetInverseJSON(sc *scene.Scene, align scene.WidgetAlignSystemType, sidStr, alignStr, widStr string, wid id.WidgetID) json.RawMessage {
	if sc == nil {
		return nil
	}
	w := sc.Widgets().Widget(wid)
	if w == nil {
		return nil
	}
	idx, loc := sc.Widgets().Alignment().System(align).Find(wid)
	inv := applyUpdateWidget{
		Kind:        "update_widget",
		SceneID:     sidStr,
		AlignSystem: alignStr,
		WidgetID:    widStr,
		Enabled:     boolPtr(w.Enabled()),
		Extended:    boolPtr(w.Extended()),
	}
	if idx >= 0 {
		inv.Index = intPtr(idx)
	}
	if loc != (scene.WidgetLocation{}) {
		inv.Location = &struct {
			Zone    string `json:"zone"`
			Section string `json:"section"`
			Area    string `json:"area"`
		}{
			Zone:    string(loc.Zone),
			Section: string(loc.Section),
			Area:    string(loc.Area),
		}
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

func boolPtr(b bool) *bool {
	v := b
	return &v
}

func intPtr(i int) *int {
	v := i
	return &v
}
