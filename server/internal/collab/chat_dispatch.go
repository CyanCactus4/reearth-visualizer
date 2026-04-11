package collab

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/reearth/reearthx/log"
)

const chatMentionsMax = 20

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
	mentions := ExtractChatMentions(m.Text, chatMentionsMax)
	msgID := uuid.NewString()
	ts := time.Now().Unix()
	chatBody := map[string]any{
		"id":     msgID,
		"userId": from.userID,
		"text":   m.Text,
		"ts":     ts,
	}
	if len(mentions) > 0 {
		chatBody["mentions"] = mentions
	}
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "chat",
		D: chatBody,
	})
	if err != nil {
		return nil
	}
	hub.fanoutRoom(ctx, from.projectID, b)
	if hub.chatStore != nil {
		pid, uid, txt, mid := from.projectID, from.userID, m.Text, msgID
		ment := mentions
		go func() {
			pctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := hub.chatStore.Append(pctx, pid, uid, txt, ts, mid, ment); err != nil {
				log.Warnfc(pctx, "collab: chat persist: %v", err)
			}
		}()
	}
	return nil
}
