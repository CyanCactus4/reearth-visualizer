package collab

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/nlslayer"
	"github.com/reearth/reearth/server/pkg/scene"
)

// buildUpdateNLSLayerInverseJSON restores fields touched by the forward update_nls_layer apply.
// Returns nil when undo cannot be represented (e.g. previous index was unset but forward set index;
// or forward changed config when the layer had no config snapshot).
func buildUpdateNLSLayerInverseJSON(layer nlslayer.NLSLayer, forward *applyUpdateNLSLayer, touchedName, touchedVis, touchedIdx, touchedCfg bool) json.RawMessage {
	if layer == nil || forward == nil || (!touchedName && !touchedVis && !touchedIdx && !touchedCfg) {
		return nil
	}
	if touchedCfg && layer.Config() == nil {
		return nil
	}
	if touchedIdx && forward.Index != nil && layer.Index() == nil {
		return nil
	}
	inv := applyUpdateNLSLayer{
		Kind:    "update_nls_layer",
		SceneID: forward.SceneID,
		LayerID: forward.LayerID,
	}
	if touchedName {
		t := layer.Title()
		inv.Name = &t
	}
	if touchedVis {
		v := layer.IsVisible()
		inv.Visible = &v
	}
	if touchedIdx && layer.Index() != nil {
		i := *layer.Index()
		inv.Index = &i
	}
	if touchedCfg && layer.Config() != nil {
		b, err := json.Marshal(map[string]any(*layer.Config()))
		if err != nil {
			return nil
		}
		inv.Config = b
	}
	if inv.Name == nil && inv.Visible == nil && inv.Index == nil && len(inv.Config) == 0 {
		return nil
	}
	b, err := json.Marshal(inv)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

func collabRunUpdateNLSLayerFromJSON(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, raw json.RawMessage) (*scene.Scene, error) {
	var p applyUpdateNLSLayer
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, err
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		return nil, err
	}
	lid, err := id.NLSLayerIDFrom(p.LayerID)
	if err != nil {
		return nil, err
	}
	hasConfig := len(p.Config) > 0 && string(p.Config) != "null"
	if p.Name == nil && p.Visible == nil && p.Index == nil && !hasConfig {
		return nil, fmt.Errorf("empty_update")
	}
	cfg, err := parseNLSConfigRaw(p.Config)
	if err != nil {
		return nil, err
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	if _, err := uc.NLSLayer.Update(opCtx, interfaces.UpdateNLSLayerInput{
		LayerID: lid,
		Index:   p.Index,
		Name:    p.Name,
		Visible: p.Visible,
		Config:  cfg,
	}, op); err != nil {
		return nil, err
	}
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, fmt.Errorf("scene reload failed")
	}
	return scenes[0], nil
}
