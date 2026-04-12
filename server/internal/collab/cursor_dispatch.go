package collab

import (
	"context"
	"encoding/json"
	"time"
)

type cursorInbound struct {
	X        *float64 `json:"x"`
	Y        *float64 `json:"y"`
	Inside   *bool    `json:"inside,omitempty"`
	ClientID string   `json:"clientId,omitempty"`
}

func dispatchCursor(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var m cursorInbound
	if err := json.Unmarshal(d, &m); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_cursor", "message": err.Error()}})
		return nil
	}
	if from.userID == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "user required for cursor"}})
		return nil
	}
	if m.X == nil || m.Y == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_cursor", "message": "x and y required"}})
		return nil
	}
	x, y := *m.X, *m.Y
	if x < 0 || x > 1 || y < 0 || y > 1 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_cursor", "message": "x,y must be in [0,1]"}})
		return nil
	}
	inside := true
	if m.Inside != nil {
		inside = *m.Inside
	}
	if !hub.cursorAllow(from.projectID, from.userID, m.ClientID) {
		return nil
	}
	dOut := map[string]any{
		"userId": from.userID,
		"x":      x,
		"y":      y,
		"inside": inside,
		"ts":     time.Now().UnixMilli(),
	}
	if m.ClientID != "" {
		dOut["clientId"] = m.ClientID
	}
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "cursor",
		D: dOut,
	})
	if err != nil {
		return nil
	}
	hub.broadcastFromClient(ctx, from.projectID, b, from)
	return nil
}
