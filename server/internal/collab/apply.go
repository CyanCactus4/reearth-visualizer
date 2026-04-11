package collab

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
)

const applyOpTimeout = 45 * time.Second

type applyEnvelope struct {
	Kind string `json:"kind"`
}

type applyUpdateWidget struct {
	Kind        string `json:"kind"`
	SceneID     string `json:"sceneId"`
	AlignSystem string `json:"alignSystem"`
	WidgetID    string `json:"widgetId"`
	Enabled     *bool  `json:"enabled,omitempty"`
	Extended    *bool  `json:"extended,omitempty"`
	Index       *int   `json:"index,omitempty"`
	Location    *struct {
		Zone    string `json:"zone"`
		Section string `json:"section"`
		Area    string `json:"area"`
	} `json:"location,omitempty"`
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

	wid, err := id.WidgetIDFrom(p.WidgetID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_widget", "message": err.Error()}})
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

	if p.Enabled == nil && p.Extended == nil && loc == nil && p.Index == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_update", "message": "no widget fields to update"}})
		return nil
	}

	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	param := interfaces.UpdateWidgetParam{
		Type:     align,
		SceneID:  sid,
		WidgetID: wid,
		Enabled:  p.Enabled,
		Extended: p.Extended,
		Location: loc,
		Index:    p.Index,
	}

	_, _, err2 := uc.Scene.UpdateWidget(opCtx, param, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}

	actor := from.userID
	if actor == "" {
		actor = "unknown"
	}
	notify := serverMessage{
		V: 1,
		T: "applied",
		D: map[string]any{
			"kind":     "update_widget",
			"sceneId":  p.SceneID,
			"widgetId": p.WidgetID,
			"userId":   actor,
		},
	}
	nb, err := json.Marshal(notify)
	if err != nil {
		return nil
	}
	hub.broadcastLocal(from.projectID, nb, from)
	from.enqueueJSON(notify)
	return nil
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
