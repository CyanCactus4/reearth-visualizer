package collab

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChatRateLimit(t *testing.T) {
	h := NewHub(Options{ChatMinIntervalMs: 500, ChatMaxRunes: 100})
	c := &Conn{hub: h, projectID: "p", userID: "alice", send: make(chan []byte, 8)}

	d1, _ := json.Marshal(map[string]string{"text": "hi"})
	err := dispatchChat(context.Background(), h, c, d1)
	assert.NoError(t, err)

	d2, _ := json.Marshal(map[string]string{"text": "again"})
	err = dispatchChat(context.Background(), h, c, d2)
	assert.NoError(t, err)
	select {
	case msg := <-c.send:
		var sm serverMessage
		_ = json.Unmarshal(msg, &sm)
		assert.Equal(t, "error", sm.T)
	default:
		t.Fatal("expected rate-limit error to sender")
	}

	time.Sleep(600 * time.Millisecond)
	d3, _ := json.Marshal(map[string]string{"text": "later"})
	err = dispatchChat(context.Background(), h, c, d3)
	assert.NoError(t, err)
}
