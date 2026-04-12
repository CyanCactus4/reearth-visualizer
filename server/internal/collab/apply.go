package collab

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
)

const applyOpTimeout = 45 * time.Second

type applyEnvelope struct {
	Kind string `json:"kind"`
}

type applyUpdateWidget struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	AlignSystem  string `json:"alignSystem"`
	WidgetID     string `json:"widgetId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
	Enabled      *bool  `json:"enabled,omitempty"`
	Extended     *bool  `json:"extended,omitempty"`
	Index        *int   `json:"index,omitempty"`
	Location     *struct {
		Zone    string `json:"zone"`
		Section string `json:"section"`
		Area    string `json:"area"`
	} `json:"location,omitempty"`
	// EntityClocks: optional per-field LWW clocks (see Hub.WidgetFieldClock). When non-empty, baseSceneRev is not required.
	EntityClocks map[string]int64 `json:"entityClocks,omitempty"`
}

type applyRemoveWidget struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	AlignSystem  string `json:"alignSystem"`
	WidgetID     string `json:"widgetId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type applyAddWidget struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	AlignSystem  string `json:"alignSystem"`
	PluginID     string `json:"pluginId"`
	ExtensionID  string `json:"extensionId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

type serverMessage struct {
	V int    `json:"v"`
	T string `json:"t"`
	D any    `json:"d,omitempty"`
}

func (c *Conn) enqueueJSON(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
	}
}

func dispatchApply(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var head applyEnvelope
	if err := json.Unmarshal(d, &head); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_json", "message": err.Error()}})
		return nil
	}

	switch head.Kind {
	case "update_widget":
		return applyUpdateWidgetOp(ctx, hub, from, d)
	case "remove_widget":
		return applyRemoveWidgetOp(ctx, hub, from, d)
	case "add_widget":
		return applyAddWidgetOp(ctx, hub, from, d)
	case "move_story_block":
		return applyMoveStoryBlockOp(ctx, hub, from, d)
	case "create_story_block":
		return applyCreateStoryBlockOp(ctx, hub, from, d)
	case "remove_story_block":
		return applyRemoveStoryBlockOp(ctx, hub, from, d)
	case "create_story_page":
		return applyCreateStoryPageOp(ctx, hub, from, d)
	case "remove_story_page":
		return applyRemoveStoryPageOp(ctx, hub, from, d)
	case "move_story_page":
		return applyMoveStoryPageOp(ctx, hub, from, d)
	case "update_story_page":
		return applyUpdateStoryPageOp(ctx, hub, from, d)
	case "duplicate_story_page":
		return applyDuplicateStoryPageOp(ctx, hub, from, d)
	case "add_nls_layer_simple":
		return applyAddNLSLayerSimpleOp(ctx, hub, from, d)
	case "remove_nls_layer":
		return applyRemoveNLSLayerOp(ctx, hub, from, d)
	case "update_nls_layer":
		return applyUpdateNLSLayerOp(ctx, hub, from, d)
	case "update_nls_layers":
		return applyUpdateNlsLayersOp(ctx, hub, from, d)
	case "create_nls_infobox":
		return applyCreateNLSInfoboxOp(ctx, hub, from, d)
	case "remove_nls_infobox":
		return applyRemoveNLSInfoboxOp(ctx, hub, from, d)
	case "create_nls_photo_overlay":
		return applyCreateNLSPhotoOverlayOp(ctx, hub, from, d)
	case "remove_nls_photo_overlay":
		return applyRemoveNLSPhotoOverlayOp(ctx, hub, from, d)
	case "add_nls_infobox_block":
		return applyAddNLSInfoboxBlockOp(ctx, hub, from, d)
	case "move_nls_infobox_block":
		return applyMoveNLSInfoboxBlockOp(ctx, hub, from, d)
	case "remove_nls_infobox_block":
		return applyRemoveNLSInfoboxBlockOp(ctx, hub, from, d)
	case "update_nls_custom_properties":
		return applyUpdateNlsCustomPropertiesOp(ctx, hub, from, d)
	case "change_nls_custom_property_title":
		return applyChangeNlsCustomPropertyTitleOp(ctx, hub, from, d)
	case "remove_nls_custom_property":
		return applyRemoveNlsCustomPropertyOp(ctx, hub, from, d)
	case "add_nls_geojson_feature":
		return applyAddNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "update_nls_geojson_feature":
		return applyUpdateNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "delete_nls_geojson_feature":
		return applyDeleteNLSGeoJSONFeatureOp(ctx, hub, from, d)
	case "add_style":
		return applyAddStyleOp(ctx, hub, from, d)
	case "update_style":
		return applyUpdateStyleOp(ctx, hub, from, d)
	case "remove_style":
		return applyRemoveStyleOp(ctx, hub, from, d)
	case "update_property_value":
		return applyUpdatePropertyValueOp(ctx, hub, from, d)
	case "merge_property_json":
		return applyMergePropertyJSONOp(ctx, hub, from, d)
	case "add_property_item":
		return applyAddPropertyItemOp(ctx, hub, from, d)
	case "remove_property_item":
		return applyRemovePropertyItemOp(ctx, hub, from, d)
	case "move_property_item":
		return applyMovePropertyItemOp(ctx, hub, from, d)
	default:
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "unknown_kind", "message": head.Kind}})
		return nil
	}
}

func applyUpdateWidgetOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyUpdateWidget
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
	clocksUsed := len(p.EntityClocks) > 0
	if !clocksUsed {
		if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
			return nil
		}
	}

	wid, err := id.WidgetIDFrom(p.WidgetID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_widget", "message": err.Error()}})
		return nil
	}
	if !widgetMustNotBeLockedByPeer(ctx, hub, from, wid) {
		return nil
	}

	align, err := parseAlignSystem(p.AlignSystem)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_align", "message": err.Error()}})
		return nil
	}

	var loc *scene.WidgetLocation
	if p.Location != nil {
		loc = &scene.WidgetLocation{
			Zone:    scene.WidgetZoneType(p.Location.Zone),
			Section: scene.WidgetSectionType(p.Location.Section),
			Area:    scene.WidgetAreaType(p.Location.Area),
		}
		if !widgetLocationValid(*loc) {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_location", "message": "invalid widget location"}})
			return nil
		}
	}

	sidStr := sid.String()
	widStr := wid.String()
	enabled := p.Enabled
	extended := p.Extended
	locUse := loc
	idxUse := p.Index
	if hub != nil && clocksUsed {
		if enabled != nil {
			if cv, ok := p.EntityClocks["enabled"]; ok && hub.WidgetFieldClock(sidStr, widStr, "enabled") > cv {
				enabled = nil
			}
		}
		if extended != nil {
			if cv, ok := p.EntityClocks["extended"]; ok && hub.WidgetFieldClock(sidStr, widStr, "extended") > cv {
				extended = nil
			}
		}
		if locUse != nil || idxUse != nil {
			if cv, ok := p.EntityClocks["layout"]; ok && hub.WidgetFieldClock(sidStr, widStr, "layout") > cv {
				locUse, idxUse = nil, nil
			}
		}
	}

	if enabled == nil && extended == nil && locUse == nil && idxUse == nil {
		code := "empty_update"
		msg := "no widget fields to update"
		if clocksUsed {
			code = "stale_entity_field"
			msg = "all touched fields are behind server entity clocks; refresh clocks from last applied"
		}
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": code, "message": msg}})
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		scenesPre, errPre := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
		if errPre == nil && len(scenesPre) > 0 {
			invJSON = buildUpdateWidgetInverseJSON(scenesPre[0], align, sidStr, p.AlignSystem, widStr, wid)
		}
	}

	param := interfaces.UpdateWidgetParam{
		Type:     align,
		SceneID:  sid,
		WidgetID: wid,
		Enabled:  enabled,
		Extended: extended,
		Location: locUse,
		Index:    idxUse,
	}

	sc, _, err2 := uc.Scene.UpdateWidget(opCtx, param, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}

	extra := map[string]any{
		"sceneId":  p.SceneID,
		"widgetId": p.WidgetID,
	}
	if hub != nil {
		fields := make([]string, 0, 4)
		if enabled != nil {
			fields = append(fields, "enabled")
		}
		if extended != nil {
			fields = append(fields, "extended")
		}
		if locUse != nil || idxUse != nil {
			fields = append(fields, "layout")
		}
		if len(fields) > 0 {
			extra["entityClocks"] = hub.BumpWidgetFieldClocks(sidStr, widStr, fields)
		}
	}

	broadcastApplied(ctx, hub, from, "update_widget", extra, sc)

	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sidStr,
			UserID:    actorUserID(from),
			Kind:      "update_widget",
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

func applyRemoveWidgetOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveWidget
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
	wid, err := id.WidgetIDFrom(p.WidgetID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_widget", "message": err.Error()}})
		return nil
	}
	if !widgetMustNotBeLockedByPeer(ctx, hub, from, wid) {
		return nil
	}
	align, err := parseAlignSystem(p.AlignSystem)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_align", "message": err.Error()}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	sidStr := sid.String()
	var invJSON json.RawMessage
	if hub != nil && hub.opStack != nil {
		scenesPre, errPre := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
		if errPre == nil && len(scenesPre) > 0 {
			invJSON = buildAddWidgetInverseJSON(scenesPre[0], sidStr, p.AlignSystem, wid)
		}
	}
	sc, err2 := uc.Scene.RemoveWidget(opCtx, align, sid, wid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	broadcastApplied(ctx, hub, from, "remove_widget", map[string]any{
		"sceneId":  p.SceneID,
		"widgetId": p.WidgetID,
	}, sc)
	if hub != nil && hub.opStack != nil && len(invJSON) > 0 {
		rec := UndoableOpRecord{
			ProjectID: from.projectID,
			SceneID:   sidStr,
			UserID:    actorUserID(from),
			Kind:      "remove_widget",
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

func applyAddWidgetOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyAddWidget
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
	align, err := parseAlignSystem(p.AlignSystem)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_align", "message": err.Error()}})
		return nil
	}
	pid, err := id.PluginIDFrom(p.PluginID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_plugin", "message": err.Error()}})
		return nil
	}
	eid := id.PluginExtensionID(p.ExtensionID)
	if string(eid) == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_extension", "message": "extensionId required"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	sc, w, err2 := uc.Scene.AddWidget(opCtx, align, sid, pid, eid, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}
	widStr := ""
	if w != nil {
		widStr = w.ID().String()
	}
	broadcastApplied(ctx, hub, from, "add_widget", map[string]any{
		"sceneId":     p.SceneID,
		"widgetId":    widStr,
		"pluginId":    p.PluginID,
		"extensionId": p.ExtensionID,
	}, sc)
	if hub != nil && hub.opStack != nil && widStr != "" {
		invJSON := buildRemoveWidgetInverseJSON(sid.String(), p.AlignSystem, widStr)
		if len(invJSON) > 0 {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "add_widget",
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

func sceneRevOf(sc *scene.Scene) int64 {
	if sc == nil {
		return 0
	}
	return sc.UpdatedAt().UnixMilli()
}

func broadcastApplied(ctx context.Context, hub *Hub, from *Conn, kind string, extra map[string]any, sc *scene.Scene) {
	_ = ctx
	d := map[string]any{
		"kind":     kind,
		"userId":   actorUserID(from),
		"sceneRev": sceneRevOf(sc),
	}
	for k, v := range extra {
		d[k] = v
	}
	notify := serverMessage{V: 1, T: "applied", D: d}
	nb, err := json.Marshal(notify)
	if err != nil {
		return
	}
	hub.broadcastLocal(from.projectID, nb, from)
	from.enqueueJSON(notify)

	sidStr := ""
	if sc != nil {
		sidStr = sc.ID().String()
	}
	if sidStr == "" {
		if s, ok := extra["sceneId"].(string); ok {
			sidStr = s
		}
	}
	rev := sceneRevOf(sc)
	if sidStr != "" && rev > 0 {
		hub.publishSceneRevision(sidStr, rev)
	}

	if hub.applyAudit != nil {
		sid, _ := extra["sceneId"].(string)
		wid, _ := extra["widgetId"].(string)
		storyID, _ := extra["storyId"].(string)
		pageID, _ := extra["pageId"].(string)
		blockID, _ := extra["blockId"].(string)
		propID, _ := extra["propertyId"].(string)
		fieldID, _ := extra["fieldId"].(string)
		styleID, _ := extra["styleId"].(string)
		layerID, _ := extra["layerId"].(string)
		var layerIDs []string
		if v, ok := extra["layerIds"].([]string); ok && len(v) > 0 {
			layerIDs = v
		}
		rec := ApplyAuditRecord{
			ProjectID:  from.projectID,
			UserID:     actorUserID(from),
			UserName:   actorUserDisplayName(from),
			Kind:       kind,
			SceneRev:   sceneRevOf(sc),
			SceneID:    sid,
			WidgetID:   wid,
			StoryID:    storyID,
			PageID:     pageID,
			BlockID:    blockID,
			PropertyID: propID,
			FieldID:    fieldID,
			StyleID:    styleID,
			LayerID:    layerID,
			LayerIDs:   layerIDs,
		}
		go func() {
			pctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := hub.applyAudit.Append(pctx, rec); err != nil {
				log.Warnfc(pctx, "collab: apply audit: %v", err)
			}
		}()
	}

	if hub != nil && sc != nil && from != nil {
		if rev := sceneRevOf(sc); rev > 0 {
			hub.queueSceneSnapshot(from, sc, rev)
		}
	}
}

func actorUserID(from *Conn) string {
	if from.userID == "" {
		return "unknown"
	}
	return from.userID
}

func actorUserDisplayName(from *Conn) string {
	if from == nil {
		return ""
	}
	if u := adapter.User(from.bgCtx); u != nil {
		if n := strings.TrimSpace(u.Name()); n != "" {
			return n
		}
	}
	return ""
}

func widgetMustNotBeLockedByPeer(ctx context.Context, hub *Hub, from *Conn, wid id.WidgetID) bool {
	holder, active, err := hub.LockHolder(ctx, from.projectID, "widget", wid.String())
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
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "object_locked", "message": "widget locked by " + holder}})
	return false
}

func parseAlignSystem(s string) (scene.WidgetAlignSystemType, error) {
	switch scene.WidgetAlignSystemType(s) {
	case scene.WidgetAlignSystemTypeDesktop, scene.WidgetAlignSystemTypeMobile:
		return scene.WidgetAlignSystemType(s), nil
	default:
		return "", fmt.Errorf("alignSystem must be desktop or mobile")
	}
}

func widgetLocationValid(l scene.WidgetLocation) bool {
	switch l.Zone {
	case scene.WidgetZoneInner, scene.WidgetZoneOuter:
	default:
		return false
	}
	switch l.Section {
	case scene.WidgetSectionLeft, scene.WidgetSectionCenter, scene.WidgetSectionRight:
	default:
		return false
	}
	switch l.Area {
	case scene.WidgetAreaTop, scene.WidgetAreaMiddle, scene.WidgetAreaBottom:
	default:
		return false
	}
	return true
}
