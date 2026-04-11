package collab

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/stretchr/testify/assert"
)

func TestCursorBroadcastToPeer(t *testing.T) {
	h := NewHub(Options{CursorMinIntervalMs: 10})
	c1 := &Conn{hub: h, projectID: "p", userID: "a", sceneID: id.SceneID{}, send: make(chan []byte, 8)}
	c2 := &Conn{hub: h, projectID: "p", userID: "b", sceneID: id.SceneID{}, send: make(chan []byte, 8)}
	h.register(c1)
	h.register(c2)
	drainConnSend(t, c1)
	drainConnSend(t, c2)

	d, _ := json.Marshal(map[string]any{"x": 0.25, "y": 0.75, "inside": true})
	err := dispatchCursor(context.Background(), h, c1, d)
	assert.NoError(t, err)

	select {
	case msg := <-c2.send:
		var sm serverMessage
		_ = json.Unmarshal(msg, &sm)
		assert.Equal(t, "cursor", sm.T)
	default:
		t.Fatal("peer should receive cursor")
	}

	select {
	case <-c1.send:
		t.Fatal("sender should not receive own cursor via fanout")
	default:
	}
}

func TestCursorRateLimitDropsExtra(t *testing.T) {
	h := NewHub(Options{CursorMinIntervalMs: 400})
	c1 := &Conn{hub: h, projectID: "p", userID: "a", sceneID: id.SceneID{}, send: make(chan []byte, 8)}
	c2 := &Conn{hub: h, projectID: "p", userID: "b", sceneID: id.SceneID{}, send: make(chan []byte, 8)}
	h.register(c1)
	h.register(c2)
	drainConnSend(t, c1)
	drainConnSend(t, c2)

	d1, _ := json.Marshal(map[string]any{"x": 0.1, "y": 0.2})
	assert.NoError(t, dispatchCursor(context.Background(), h, c1, d1))
	<-c2.send

	d2, _ := json.Marshal(map[string]any{"x": 0.3, "y": 0.4})
	assert.NoError(t, dispatchCursor(context.Background(), h, c1, d2))
	select {
	case <-c2.send:
		t.Fatal("second cursor within window should be dropped")
	case <-time.After(80 * time.Millisecond):
	}

	time.Sleep(450 * time.Millisecond)
	d3, _ := json.Marshal(map[string]any{"x": 0.5, "y": 0.6})
	assert.NoError(t, dispatchCursor(context.Background(), h, c1, d3))
	select {
	case <-c2.send:
	default:
		t.Fatal("expected cursor after interval")
	}
}
