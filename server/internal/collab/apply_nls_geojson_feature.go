package collab

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearthx/log"
)

type applyAddNLSGeoJSONFeature struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	Type         string          `json:"type"`
	Geometry     json.RawMessage `json:"geometry"`
	Properties   json.RawMessage `json:"properties,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyUpdateNLSGeoJSONFeature struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	FeatureID    string          `json:"featureId"`
	Geometry     json.RawMessage `json:"geometry,omitempty"`
	Properties   json.RawMessage `json:"properties,omitempty"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyDeleteNLSGeoJSONFeature struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	LayerID      string `json:"layerId"`
	FeatureID    string `json:"featureId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func geoJSONObjectMapRequired(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("geometry is required")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, fmt.Errorf("geometry must be an object")
	}
	return m, nil
}

func geoJSONPropertiesPtr(raw json.RawMessage) (*map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func geoJSONOptionalObjectMapPtr(raw json.RawMessage) (*map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func applyAddNLSGeoJSONFeatureOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddNLSGeoJSONFeature
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
	if p.Type == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "type required"}})
		return nil
	}
	geomMap, err := geoJSONObjectMapRequired(p.Geometry)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	propsPtr, err := geoJSONPropertiesPtr(p.Properties)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	feat, err2 := uc.NLSLayer.AddGeoJSONFeature(opCtx, interfaces.AddNLSLayerGeoJSONFeatureParams{
		LayerID:    lid,
		Type:       p.Type,
		Geometry:   geomMap,
		Properties: propsPtr,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "add_nls_geojson_feature", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil {
		inv := applyDeleteNLSGeoJSONFeature{
			Kind:      "delete_nls_geojson_feature",
			SceneID:   p.SceneID,
			LayerID:   p.LayerID,
			FeatureID: feat.ID().String(),
		}
		if invJSON, errM := json.Marshal(inv); errM == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_nls_geojson_feature",
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

func applyUpdateNLSGeoJSONFeatureOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateNLSGeoJSONFeature
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
	fid, err := id.FeatureIDFrom(p.FeatureID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "featureId: " + err.Error()}})
		return nil
	}
	geomPtr, err := geoJSONOptionalObjectMapPtr(p.Geometry)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	propsPtr, err := geoJSONOptionalObjectMapPtr(p.Properties)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	if geomPtr == nil && propsPtr == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "geometry or properties required"}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		preCtx, preCancel := context.WithTimeout(ctx, applyOpTimeout)
		listL, errL := uc.NLSLayer.Fetch(preCtx, id.NLSLayerIDList{lid}, op)
		preCancel()
		if errL == nil && len(listL) > 0 && listL[0] != nil {
			ly := *listL[0]
			fptr, errF := findNLSFeature(ly, fid)
			if errF == nil && fptr != nil {
				inv := applyUpdateNLSGeoJSONFeature{
					Kind:      "update_nls_geojson_feature",
					SceneID:   p.SceneID,
					LayerID:   p.LayerID,
					FeatureID: p.FeatureID,
				}
				if len(p.Geometry) > 0 {
					if gm, errG := geometryToGeoJSONMap(fptr.Geometry()); errG == nil {
						if gb, errM := json.Marshal(gm); errM == nil {
							inv.Geometry = gb
						}
					}
				}
				if len(p.Properties) > 0 && fptr.Properties() != nil {
					if pb, errM := json.Marshal(*fptr.Properties()); errM == nil {
						inv.Properties = pb
					}
				}
				if len(inv.Geometry) > 0 || len(inv.Properties) > 0 {
					invJSON, _ = json.Marshal(inv)
				}
			}
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.UpdateGeoJSONFeature(opCtx, interfaces.UpdateNLSLayerGeoJSONFeatureParams{
		LayerID:    lid,
		FeatureID:  fid,
		Geometry:   geomPtr,
		Properties: propsPtr,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "update_nls_geojson_feature", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "update_nls_geojson_feature",
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

func applyDeleteNLSGeoJSONFeatureOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyDeleteNLSGeoJSONFeature
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
	fid, err := id.FeatureIDFrom(p.FeatureID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": "featureId: " + err.Error()}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		preCtx, preCancel := context.WithTimeout(ctx, applyOpTimeout)
		listL, errL := uc.NLSLayer.Fetch(preCtx, id.NLSLayerIDList{lid}, op)
		preCancel()
		if errL == nil && len(listL) > 0 && listL[0] != nil {
			ly := *listL[0]
			fptr, errF := findNLSFeature(ly, fid)
			if errF == nil && fptr != nil {
				gm, errG := geometryToGeoJSONMap(fptr.Geometry())
				if errG == nil {
					gb, errM := json.Marshal(gm)
					if errM == nil {
						inv := applyAddNLSGeoJSONFeature{
							Kind:     "add_nls_geojson_feature",
							SceneID:  p.SceneID,
							LayerID:  p.LayerID,
							Type:     fptr.FeatureType(),
							Geometry: gb,
						}
						if fptr.Properties() != nil {
							if pb, errP := json.Marshal(*fptr.Properties()); errP == nil {
								inv.Properties = pb
							}
						}
						invJSON, _ = json.Marshal(inv)
					}
				}
			}
		}
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.DeleteGeoJSONFeature(opCtx, interfaces.DeleteNLSLayerGeoJSONFeatureParams{
		LayerID:   lid,
		FeatureID: fid,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "delete_nls_geojson_feature", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "delete_nls_geojson_feature",
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
