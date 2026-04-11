package collab

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// notifyMentionWebhook posts a JSON payload to MentionWebhookURL (best-effort, async).
func (h *Hub) notifyMentionWebhook(ctx context.Context, payload map[string]any) {
	if h == nil || h.mentionWebhook == "" || len(payload) == 0 {
		return
	}
	url := h.mentionWebhook
	go func() {
		pctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		b, err := json.Marshal(payload)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(pctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		_, _ = http.DefaultClient.Do(req)
	}()
}
