package collab

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/nlslayer"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
)

type applyAddNLSLayerSimple struct {
	Kind           string          `json:"kind"`
	SceneID        string          `json:"sceneId"`
	Title          string          `json:"title"`
	LayerType      string          `json:"layerType"`
	Config         json.RawMessage `json:"config,omitempty"`
	Index          *int            `json:"index,omitempty"`
	Visible        *bool           `json:"visible,omitempty"`
	Schema         json.RawMessage `json:"schema,omitempty"`
	DataSourceName *string         `json:"dataSourceName,omitempty"`
	BaseSceneRev   *int64          `json:"baseSceneRev,omitempty"`
}

type applyRemoveNLSLayer struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyUpdateNLSLayer struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	Index        *int            `json:"index,omitempty"`
	Name         *string         `json:"name,omitempty"`
	Visible      *bool           `json:"visible,omitempty"`
	Config       json.RawMessage `json:"config,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyUpdateNlsLayers struct {
	Kind         string                    `json:"kind"`
	SceneID      string                    `json:"sceneId"`
	Layers       []applyUpdateNLSLayerItem `json:"layers"`
	BaseSceneRev *int64                    `json:"baseSceneRev,omitempty"`
}

type applyUpdateNLSLayerItem struct {
	LayerID string          `json:"layerId"`
	Index   *int            `json:"index,omitempty"`
	Name    *string         `json:"name,omitempty"`
	Visible *bool           `json:"visible,omitempty"`
	Config  json.RawMessage `json:"config,omitempty"`
}

func nlsLayerMustNotBeLockedByPeer(ctx context.Context, hub *Hub, from *Conn, lid id.NLSLayerID) bool {
	holder, active, err := hub.LockHolder(ctx, from.projectID, "layer", lid.String())
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
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "object_locked", "message": "layer locked by " + holder}})
	return false
}

func parseNLSConfigRaw(raw json.RawMessage) (*nlslayer.Config, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	c := nlslayer.Config(m)
	return &c, nil
}

func parseNLSSchemaRaw(raw json.RawMessage) (*map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func nlsLayerBelongsToScene(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, lid id.NLSLayerID, from *Conn) bool {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.NLSLayer.Fetch(opCtx, id.NLSLayerIDList{lid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": "layer not found"}})
		return false
	}
	if (*list[0]).Scene() != sid {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "layer does not belong to this scene"}})
		return false
	}
	return true
}

func fetchSceneAfterNLSChange(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, from *Conn) *scene.Scene {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	return scenes[0]
}

// fetchSceneAfterNLSSilent reloads the scene after an NLS mutation (REST undo / tests; no WS error enqueue).
func fetchSceneAfterNLSSilent(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID) (*scene.Scene, error) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		return nil, fmt.Errorf("scene reload failed")
	}
	return scenes[0], nil
}

func applyAddNLSLayerSimpleOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddNLSLayerSimple
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
	lt, err := nlslayer.NewLayerType(p.LayerType)
	if err != nil || !lt.IsValidLayerType() {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer_type", "message": "layerType must be simple or group"}})
		return nil
	}
	cfg, err := parseNLSConfigRaw(p.Config)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	schema, err := parseNLSSchemaRaw(p.Schema)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	layer, err2 := uc.NLSLayer.AddLayerSimple(opCtx, interfaces.AddNLSLayerSimpleInput{
		SceneID:        sid,
		Title:          p.Title,
		Index:          p.Index,
		LayerType:      lt,
		Config:         cfg,
		Visible:        p.Visible,
		Schema:         schema,
		DataSourceName: p.DataSourceName,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	lidStr := ""
	if layer != nil {
		lidStr = layer.ID().String()
	}
	broadcastApplied(ctx, hub, from, "add_nls_layer_simple", map[string]any{
		"sceneId": p.SceneID,
		"layerId": lidStr,
	}, sc)
	if hub != nil && hub.opStack != nil && layer != nil {
		inv := applyRemoveNLSLayer{
			Kind:    "remove_nls_layer",
			SceneID: p.SceneID,
			LayerID: layer.ID().String(),
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_nls_layer_simple",
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

func applyRemoveNLSLayerOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveNLSLayer
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
	lid, err := id.NLSLayerIDFrom(p.LayerID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": err.Error()}})
		return nil
	}
	if !nlsLayerBelongsToScene(ctx, uc, op, sid, lid, from) {
		return nil
	}
	if !nlsLayerMustNotBeLockedByPeer(ctx, hub, from, lid) {
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		preCtx, preCancel := context.WithTimeout(ctx, applyOpTimeout)
		listL, errL := uc.NLSLayer.Fetch(preCtx, id.NLSLayerIDList{lid}, op)
		preCancel()
		if errL == nil && len(listL) > 0 && listL[0] != nil {
			ly := *listL[0]
			if sl := nlslayer.NLSLayerSimpleFromLayer(ly); sl != nil {
				if raw, errM := marshalAddNLSLayerSimpleJSONFromSimple(p.SceneID, sl); errM == nil {
					invJSON = raw
				}
			}
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, err2 := uc.NLSLayer.Remove(opCtx, lid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_nls_layer", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "remove_nls_layer",
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

func applyUpdateNLSLayerOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateNLSLayer
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
	lid, err := id.NLSLayerIDFrom(p.LayerID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": err.Error()}})
		return nil
	}
	if !nlsLayerBelongsToScene(ctx, uc, op, sid, lid, from) {
		return nil
	}
	if !nlsLayerMustNotBeLockedByPeer(ctx, hub, from, lid) {
		return nil
	}
	hasConfig := len(p.Config) > 0 && string(p.Config) != "null"
	if p.Name == nil && p.Visible == nil && p.Index == nil && !hasConfig {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "no layer fields to update"}})
		return nil
	}
	cfg, err := parseNLSConfigRaw(p.Config)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	touchedName := p.Name != nil
	touchedVis := p.Visible != nil
	touchedIdx := p.Index != nil
	touchedCfg := hasConfig
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		prevList, errPre := uc.NLSLayer.Fetch(opCtx, id.NLSLayerIDList{lid}, op)
		if errPre == nil && len(prevList) > 0 && prevList[0] != nil {
			prev := *prevList[0]
			invJSON = buildUpdateNLSLayerInverseJSON(prev, &p, touchedName, touchedVis, touchedIdx, touchedCfg)
		}
	}

	_, err2 := uc.NLSLayer.Update(opCtx, interfaces.UpdateNLSLayerInput{
		LayerID: lid,
		Index:   p.Index,
		Name:    p.Name,
		Visible: p.Visible,
		Config:  cfg,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "update_nls_layer", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "update_nls_layer",
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

func applyUpdateNlsLayersOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateNlsLayers
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
	if len(p.Layers) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "layers required"}})
		return nil
	}
	for _, row := range p.Layers {
		lid, errL := id.NLSLayerIDFrom(row.LayerID)
		if errL != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": errL.Error()}})
			return nil
		}
		if !nlsLayerBelongsToScene(ctx, uc, op, sid, lid, from) {
			return nil
		}
		if !nlsLayerMustNotBeLockedByPeer(ctx, hub, from, lid) {
			return nil
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	wantStack := hub != nil && hub.opStack != nil
	var invPieces []applyUpdateNLSLayerItem

	for _, row := range p.Layers {
		lid, errL := id.NLSLayerIDFrom(row.LayerID)
		if errL != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_layer", "message": errL.Error()}})
			return nil
		}
		rowHasCfg := len(row.Config) > 0 && string(row.Config) != "null"
		if row.Name == nil && row.Visible == nil && row.Index == nil && !rowHasCfg {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "no layer fields to update"}})
			return nil
		}
		cfg, errC := parseNLSConfigRaw(row.Config)
		if errC != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": errC.Error()}})
			return nil
		}

		if wantStack {
			prevList, errPre := uc.NLSLayer.Fetch(opCtx, id.NLSLayerIDList{lid}, op)
			if errPre != nil || len(prevList) == 0 || prevList[0] == nil {
				wantStack = false
				invPieces = nil
			} else {
				prev := *prevList[0]
				fwd := applyUpdateNLSLayer{
					SceneID: p.SceneID,
					LayerID: row.LayerID,
					Name:    row.Name,
					Visible: row.Visible,
					Index:   row.Index,
					Config:  row.Config,
				}
				touchedName := row.Name != nil
				touchedVis := row.Visible != nil
				touchedIdx := row.Index != nil
				touchedCfg := rowHasCfg
				invRaw := buildUpdateNLSLayerInverseJSON(prev, &fwd, touchedName, touchedVis, touchedIdx, touchedCfg)
				if len(invRaw) == 0 {
					wantStack = false
					invPieces = nil
				} else {
					it, errI := itemFromUpdateNLSInverseRaw(invRaw)
					if errI != nil {
						wantStack = false
						invPieces = nil
					} else {
						invPieces = append(invPieces, it)
					}
				}
			}
		}

		_, errU := uc.NLSLayer.Update(opCtx, interfaces.UpdateNLSLayerInput{
			LayerID: lid,
			Index:   row.Index,
			Name:    row.Name,
			Visible: row.Visible,
			Config:  cfg,
		}, op)
		if errU != nil {
			wantStack = false
			invPieces = nil
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": errU.Error()}})
			return nil
		}
	}

	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	ids := make([]string, 0, len(p.Layers))
	for _, row := range p.Layers {
		ids = append(ids, row.LayerID)
	}
	broadcastApplied(ctx, hub, from, "update_nls_layers", map[string]any{
		"sceneId":  p.SceneID,
		"layerIds": ids,
	}, sc)

	if wantStack && len(invPieces) == len(p.Layers) {
		reverseUpdateNLSLayerItems(invPieces)
		inv := applyUpdateNlsLayers{
			Kind:    "update_nls_layers",
			SceneID: p.SceneID,
			Layers:  invPieces,
		}
		binv, errI := json.Marshal(inv)
		if errI == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "update_nls_layers",
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

// marshalAddNLSLayerSimpleJSONFromSimple builds an add_nls_layer_simple apply body from a simple layer snapshot
// (inverse of remove_nls_layer). Infobox / photo overlay / sketch feature data beyond schema are not serialized here.
func marshalAddNLSLayerSimpleJSONFromSimple(sceneID string, sl *nlslayer.NLSLayerSimple) (json.RawMessage, error) {
	if sl == nil {
		return nil, fmt.Errorf("nil layer")
	}
	inv := applyAddNLSLayerSimple{
		Kind:      "add_nls_layer_simple",
		SceneID:   sceneID,
		Title:     sl.Title(),
		LayerType: string(sl.LayerType()),
	}
	if sl.Index() != nil {
		i := *sl.Index()
		inv.Index = &i
	}
	v := sl.IsVisible()
	inv.Visible = &v
	if cfg := sl.Config(); cfg != nil {
		m := map[string]any(*cfg)
		b, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		inv.Config = json.RawMessage(b)
	}
	if ds := sl.DataSourceName(); ds != nil {
		s := *ds
		inv.DataSourceName = &s
	}
	if sk := sl.Sketch(); sk != nil && sk.CustomPropertySchema() != nil {
		b, err := json.Marshal(*sk.CustomPropertySchema())
		if err != nil {
			return nil, err
		}
		inv.Schema = json.RawMessage(b)
	}
	return json.Marshal(inv)
}
