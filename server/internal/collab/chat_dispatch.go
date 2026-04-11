package collab

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"
)

type chatInbound struct {
	Text string `json:"text"`
}

func dispatchChat(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var m chatInbound
	if err := json.Unmarshal(d, &m); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_chat", "message": err.Error()}})
		return nil
	}
	if from.userID == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "user required for chat"}})
		return nil
	}
	n := utf8.RuneCountInString(m.Text)
	if n == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "empty_chat", "message": "text required"}})
		return nil
	}
	if n > hub.chatMaxRunes {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "chat_too_long", "message": "message too long"}})
		return nil
	}
	if !hub.chatAllow(from.projectID, from.userID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "chat_rate", "message": "rate limit exceeded"}})
		return nil
	}
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "chat",
		D: map[string]any{
			"userId": from.userID,
			"text":   m.Text,
			"ts":     time.Now().Unix(),
		},
	})
	if err != nil {
		return nil
	}
	hub.fanoutRoom(ctx, from.projectID, b)
	return nil
}
