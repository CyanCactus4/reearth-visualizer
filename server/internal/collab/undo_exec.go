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
	"github.com/reearth/reearth/server/pkg/nlslayer"
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
		if operator == nil || !operator.IsWritableScene(sid) {
			return nil, errors.New("write not allowed")
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
	case "add_widget":
		var p applyAddWidget
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
		align, err := parseAlignSystem(p.AlignSystem)
		if err != nil {
			return nil, err
		}
		pid, err := id.PluginIDFrom(p.PluginID)
		if err != nil {
			return nil, err
		}
		eid := id.PluginExtensionID(p.ExtensionID)
		if string(eid) == "" {
			return nil, fmt.Errorf("extensionId required")
		}
		sc, _, err := uc.Scene.AddWidget(opCtx, align, sid, pid, eid, operator)
		return sc, err
	case "remove_widget":
		var p applyRemoveWidget
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
		wid, err := id.WidgetIDFrom(p.WidgetID)
		if err != nil {
			return nil, err
		}
		align, err := parseAlignSystem(p.AlignSystem)
		if err != nil {
			return nil, err
		}
		sc, err := uc.Scene.RemoveWidget(opCtx, align, sid, wid, operator)
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
		if operator == nil || !operator.IsWritableScene(sid) {
			return nil, errors.New("write not allowed")
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
	case "move_story_page":
		var p applyMoveStoryPage
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
		return collabRunMoveStoryPageFromJSON(ctx, uc, operator, raw)
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
	case "add_property_item":
		var p applyAddPropertyItem
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
		pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
		if err != nil {
			return nil, err
		}
		sc, _, err := runAddPropertyItemCore(ctx, uc, operator, sid, pid, &p)
		return sc, err
	case "remove_property_item":
		var p applyRemovePropertyItem
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
		pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
		if err != nil {
			return nil, err
		}
		return runRemovePropertyItemCore(ctx, uc, operator, sid, pid, &p)
	case "move_property_item":
		var p applyMovePropertyItem
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
		pid, err := gqlmodel.ToID[id.Property](gqlmodel.ID(p.PropertyID))
		if err != nil {
			return nil, err
		}
		return runMovePropertyItemCore(ctx, uc, operator, sid, pid, &p)
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
	case "add_style":
		var p applyAddStyle
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
		val, err := parseStyleValueRaw(p.Value)
		if err != nil {
			return nil, err
		}
		if val == nil {
			return nil, fmt.Errorf("value required")
		}
		if _, err := uc.Style.AddStyle(opCtx, interfaces.AddStyleInput{
			SceneID: sid,
			Name:    p.Name,
			Value:   val,
		}, operator); err != nil {
			return nil, err
		}
		scenes, err2 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, operator)
		if err2 != nil || len(scenes) == 0 {
			return nil, fmt.Errorf("scene reload failed")
		}
		return scenes[0], nil
	case "remove_style":
		var p applyRemoveStyle
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
		stid, err := id.StyleIDFrom(p.StyleID)
		if err != nil {
			return nil, err
		}
		if _, err := uc.Style.RemoveStyle(opCtx, stid, operator); err != nil {
			return nil, err
		}
		scenes, err2 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, operator)
		if err2 != nil || len(scenes) == 0 {
			return nil, fmt.Errorf("scene reload failed")
		}
		return scenes[0], nil
	case "update_nls_layer":
		var p applyUpdateNLSLayer
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
		return collabRunUpdateNLSLayerFromJSON(ctx, uc, operator, raw)
	case "update_nls_layers":
		var p applyUpdateNlsLayers
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
		return collabRunUpdateNlsLayersFromJSON(ctx, uc, operator, raw)
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
	case "update_nls_custom_properties":
		var p applyUpdateNlsCustomProperties
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.AddOrUpdateCustomProperties(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
			LayerID: lid,
			Schema:  schemaMap,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "change_nls_custom_property_title":
		var p applyChangeNlsCustomPropertyTitle
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.ChangeCustomPropertyTitle(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
			LayerID: lid,
			Schema:  schemaMap,
		}, p.OldTitle, p.NewTitle, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "restore_nls_custom_property_removed":
		var p applyRestoreNlsCustomPropertyRemoved
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		schemaMap, err := nlsCustomPropertySchemaMap(p.Schema)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.AddOrUpdateCustomProperties(opCtx, interfaces.AddOrUpdateCustomPropertiesInput{
			LayerID: lid,
			Schema:  schemaMap,
		}, operator)
		if err != nil {
			return nil, err
		}
		for _, fv := range p.FeatureValues {
			fid, err := id.FeatureIDFrom(fv.FeatureID)
			if err != nil {
				return nil, err
			}
			list, err2 := uc.NLSLayer.Fetch(opCtx, id.NLSLayerIDList{lid}, operator)
			if err2 != nil || len(list) == 0 || list[0] == nil {
				return nil, fmt.Errorf("nls layer fetch failed")
			}
			layer := *list[0]
			feat, err2 := findNLSFeature(layer, fid)
			if err2 != nil {
				return nil, err2
			}
			merged := map[string]any{}
			if fp := feat.Properties(); fp != nil && *fp != nil {
				for k, v := range *fp {
					merged[k] = v
				}
			}
			var val any
			if len(fv.Value) > 0 && string(fv.Value) != "null" {
				if err2 := json.Unmarshal(fv.Value, &val); err2 != nil {
					return nil, err2
				}
			}
			merged[p.RemovedTitle] = val
			_, err = uc.NLSLayer.UpdateGeoJSONFeature(opCtx, interfaces.UpdateNLSLayerGeoJSONFeatureParams{
				LayerID:    lid,
				FeatureID:  fid,
				Geometry:   nil,
				Properties: &merged,
			}, operator)
			if err != nil {
				return nil, err
			}
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "add_nls_geojson_feature":
		var p applyAddNLSGeoJSONFeature
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		geomMap, err := geoJSONObjectMapRequired(p.Geometry)
		if err != nil {
			return nil, err
		}
		propsPtr, err := geoJSONPropertiesPtr(p.Properties)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.AddGeoJSONFeature(opCtx, interfaces.AddNLSLayerGeoJSONFeatureParams{
			LayerID:    lid,
			Type:       p.Type,
			Geometry:   geomMap,
			Properties: propsPtr,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "update_nls_geojson_feature":
		var p applyUpdateNLSGeoJSONFeature
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		fid, err := id.FeatureIDFrom(p.FeatureID)
		if err != nil {
			return nil, err
		}
		geomPtr, err := geoJSONOptionalObjectMapPtr(p.Geometry)
		if err != nil {
			return nil, err
		}
		propsPtr, err := geoJSONOptionalObjectMapPtr(p.Properties)
		if err != nil {
			return nil, err
		}
		if geomPtr == nil && propsPtr == nil {
			return nil, fmt.Errorf("geometry or properties required")
		}
		_, err = uc.NLSLayer.UpdateGeoJSONFeature(opCtx, interfaces.UpdateNLSLayerGeoJSONFeatureParams{
			LayerID:    lid,
			FeatureID:  fid,
			Geometry:   geomPtr,
			Properties: propsPtr,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "delete_nls_geojson_feature":
		var p applyDeleteNLSGeoJSONFeature
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		fid, err := id.FeatureIDFrom(p.FeatureID)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.DeleteGeoJSONFeature(opCtx, interfaces.DeleteNLSLayerGeoJSONFeatureParams{
			LayerID:   lid,
			FeatureID: fid,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "create_nls_infobox":
		var p applyCreateNLSInfobox
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.CreateNLSInfobox(opCtx, lid, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "remove_nls_infobox":
		var p applyRemoveNLSInfobox
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.RemoveNLSInfobox(opCtx, lid, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "create_nls_photo_overlay":
		var p applyCreateNLSPhotoOverlay
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.CreateNLSPhotoOverlay(opCtx, lid, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "remove_nls_photo_overlay":
		var p applyRemoveNLSPhotoOverlay
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.RemoveNLSPhotoOverlay(opCtx, lid, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "add_nls_infobox_block":
		var p applyAddNLSInfoboxBlock
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		pid, err := id.PluginIDFrom(p.PluginID)
		if err != nil {
			return nil, err
		}
		eid := id.PluginExtensionID(p.ExtensionID)
		if string(eid) == "" {
			return nil, fmt.Errorf("extensionId required")
		}
		_, _, err = uc.NLSLayer.AddNLSInfoboxBlock(opCtx, interfaces.AddNLSInfoboxBlockParam{
			LayerID:     lid,
			PluginID:    pid,
			ExtensionID: eid,
			Index:       p.Index,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "remove_nls_infobox_block":
		var p applyRemoveNLSInfoboxBlock
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		bid, err := id.InfoboxBlockIDFrom(p.InfoboxBlockID)
		if err != nil {
			return nil, err
		}
		_, _, err = uc.NLSLayer.RemoveNLSInfoboxBlock(opCtx, interfaces.RemoveNLSInfoboxBlockParam{
			LayerID:        lid,
			InfoboxBlockID: bid,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "move_nls_infobox_block":
		var p applyMoveNLSInfoboxBlock
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		bid, err := id.InfoboxBlockIDFrom(p.InfoboxBlockID)
		if err != nil {
			return nil, err
		}
		_, _, _, err = uc.NLSLayer.MoveNLSInfoboxBlock(opCtx, interfaces.MoveNLSInfoboxBlockParam{
			LayerID:        lid,
			InfoboxBlockID: bid,
			Index:          p.Index,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "add_nls_layer_simple":
		var p applyAddNLSLayerSimple
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
		lt, err := nlslayer.NewLayerType(p.LayerType)
		if err != nil || !lt.IsValidLayerType() {
			return nil, fmt.Errorf("invalid layerType")
		}
		cfg, err := parseNLSConfigRaw(p.Config)
		if err != nil {
			return nil, err
		}
		schema, err := parseNLSSchemaRaw(p.Schema)
		if err != nil {
			return nil, err
		}
		_, err = uc.NLSLayer.AddLayerSimple(opCtx, interfaces.AddNLSLayerSimpleInput{
			SceneID:        sid,
			Title:          p.Title,
			Index:          p.Index,
			LayerType:      lt,
			Config:         cfg,
			Visible:        p.Visible,
			Schema:         schema,
			DataSourceName: p.DataSourceName,
		}, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	case "remove_nls_layer":
		var p applyRemoveNLSLayer
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
		lid, err := id.NLSLayerIDFrom(p.LayerID)
		if err != nil {
			return nil, err
		}
		_, _, err = uc.NLSLayer.Remove(opCtx, lid, operator)
		if err != nil {
			return nil, err
		}
		return fetchSceneAfterNLSSilent(ctx, uc, operator, sid)
	default:
		return nil, fmt.Errorf("unsupported undo kind %q", env.Kind)
	}
}
