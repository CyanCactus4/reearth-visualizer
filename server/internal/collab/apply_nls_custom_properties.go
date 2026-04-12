package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearthx/log"
)

type applyUpdateNlsCustomProperties struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	Schema       json.RawMessage `json:"schema"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyChangeNlsCustomPropertyTitle struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	Schema       json.RawMessage `json:"schema"`
	OldTitle     string          `json:"oldTitle"`
	NewTitle     string          `json:"newTitle"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type applyRemoveNlsCustomProperty struct {
	Kind         string          `json:"kind"`
	SceneID      string          `json:"sceneId"`
	LayerID      string          `json:"layerId"`
	Schema       json.RawMessage `json:"schema"`
	RemovedTitle string          `json:"removedTitle"`
	BaseSceneRev *int64          `json:"baseSceneRev,omitempty"`
}

type nlsCustomPropertyRemovedFeatureValue struct {
	FeatureID string          `json:"featureId"`
	Value     json.RawMessage `json:"value"`
}

// applyRestoreNlsCustomPropertyRemoved is server-only undo inverse for remove_nls_custom_property.
type applyRestoreNlsCustomPropertyRemoved struct {
	Kind          string                                 `json:"kind"`
	SceneID       string                                 `json:"sceneId"`
	LayerID       string                                 `json:"layerId"`
	Schema        json.RawMessage                        `json:"schema"`
	RemovedTitle  string                                 `json:"removedTitle"`
	FeatureValues []nlsCustomPropertyRemovedFeatureValue `json:"featureValues"`
}

func nlsCustomPropertySchemaMap(raw json.RawMessage) (map[string]any, error) {
	pm, err := parseNLSSchemaRaw(raw)
	if err != nil {
		return nil, err
	}
	if pm == nil {
		return map[string]any{}, nil
	}
	return *pm, nil
}

func applyUpdateNlsCustomPropertiesOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateNlsCustomProperties
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
	schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.AddOrUpdateCustomProperties(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
		LayerID: lid,
		Schema:  schemaMap,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "update_nls_custom_properties", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	return nil
}

func applyChangeNlsCustomPropertyTitleOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyChangeNlsCustomPropertyTitle
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
		inv := applyChangeNlsCustomPropertyTitle{
			Kind:     "change_nls_custom_property_title",
			SceneID:  p.SceneID,
			LayerID:  p.LayerID,
			Schema:   p.Schema,
			OldTitle: p.NewTitle,
			NewTitle: p.OldTitle,
		}
		invJSON, _ = json.Marshal(inv)
	}
	schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.ChangeCustomPropertyTitle(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
		LayerID: lid,
		Schema:  schemaMap,
	}, p.OldTitle, p.NewTitle, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "change_nls_custom_property_title", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "change_nls_custom_property_title",
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

func applyRemoveNlsCustomPropertyOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveNlsCustomProperty
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
	schemaBefore, err := nlsLayerCustomSchemaRaw(ctx, uc, op, lid)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err.Error()}})
		return nil
	}
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		snapCtx, snapCancel := context.WithTimeout(ctx, applyOpTimeout)
		featureVals, snapErr := nlsCustomPropertyFeatureValuesForTitle(snapCtx, uc, op, lid, p.RemovedTitle)
		snapCancel()
		if snapErr != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": snapErr.Error()}})
			return nil
		}
		inv := applyRestoreNlsCustomPropertyRemoved{
			Kind:          "restore_nls_custom_property_removed",
			SceneID:       p.SceneID,
			LayerID:       p.LayerID,
			Schema:        schemaBefore,
			RemovedTitle:  p.RemovedTitle,
			FeatureValues: featureVals,
		}
		invJSON, _ = json.Marshal(inv)
	}
	schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, err2 := uc.NLSLayer.RemoveCustomProperty(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
		LayerID: lid,
		Schema:  schemaMap,
	}, p.RemovedTitle, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	sc := fetchSceneAfterNLSChange(ctx, uc, op, sid, from)
	if sc == nil {
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_nls_custom_property", map[string]any{
		"sceneId": p.SceneID,
		"layerId": p.LayerID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sid.String(),
			UserID:    actorUserID(from),
			Kind:      "remove_nls_custom_property",
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

func nlsCustomPropertyFeatureValuesForTitle(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, lid id.NLSLayerID, title string) ([]nlsCustomPropertyRemovedFeatureValue, error) {
	list, err := uc.NLSLayer.Fetch(ctx, id.NLSLayerIDList{lid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		return nil, err
	}
	layer := *list[0]
	sk := layer.Sketch()
	if sk == nil || sk.FeatureCollection() == nil {
		return nil, nil
	}
	var out []nlsCustomPropertyRemovedFeatureValue
	for _, f := range sk.FeatureCollection().Features() {
		props := f.Properties()
		if props == nil || *props == nil {
			continue
		}
		v, ok := (*props)[title]
		if !ok {
			continue
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		out = append(out, nlsCustomPropertyRemovedFeatureValue{
			FeatureID: f.ID().String(),
			Value:     json.RawMessage(raw),
		})
	}
	return out, nil
}

func nlsLayerCustomSchemaRaw(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, lid id.NLSLayerID) (json.RawMessage, error) {
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	list, err := uc.NLSLayer.Fetch(opCtx, id.NLSLayerIDList{lid}, op)
	if err != nil || len(list) == 0 || list[0] == nil {
		return nil, err
	}
	layer := *list[0]
	sk := layer.Sketch()
	if sk == nil || sk.CustomPropertySchema() == nil {
		return json.RawMessage("{}"), nil
	}
	b, err := json.Marshal(*sk.CustomPropertySchema())
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}
