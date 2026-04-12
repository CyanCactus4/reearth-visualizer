package collab

import (
	"context"
	"encoding/json"
	"time"
)

type activityInbound struct {
	Kind     string `json:"kind"`
	ClientID string `json:"clientId,omitempty"`
}

func dispatchActivity(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var m activityInbound
	if err := json.Unmarshal(d, &m); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_activity", "message": err.Error()}})
		return nil
	}
	if from.userID == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "user required for activity"}})
		return nil
	}
	switch m.Kind {
	case "typing", "move":
	default:
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_activity", "message": "unknown kind"}})
		return nil
	}
	if !hub.activityAllow(from.projectID, from.userID, m.ClientID, m.Kind) {
		return nil
	}
	dOut := map[string]any{
		"userId": from.userID,
		"kind":   m.Kind,
		"ts":     time.Now().UnixMilli(),
	}
	if m.ClientID != "" {
		dOut["clientId"] = m.ClientID
	}
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "activity",
		D: dOut,
	})
	if err != nil {
		return nil
	}
	hub.broadcastFromClient(ctx, from.projectID, b, from)
	return nil
}
