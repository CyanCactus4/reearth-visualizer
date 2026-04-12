package collab

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub_NoRedis(t *testing.T) {
	h := NewHub(Options{})
	require.NotNil(t, h)
	assert.Nil(t, h.relay)
}

func drainConnSend(t *testing.T, c *Conn) {
	t.Helper()
	for {
		select {
		case <-c.send:
		default:
			return
		}
	}
}

func TestHubRegisterBroadcastUnregister(t *testing.T) {
	h := NewHub(Options{})
	c1 := &Conn{hub: h, projectID: "proj-1", send: make(chan []byte, 16)}
	c2 := &Conn{hub: h, projectID: "proj-1", send: make(chan []byte, 16)}

	h.register(c1)
	drainConnSend(t, c1)
	h.register(c2)
	drainConnSend(t, c1)
	drainConnSend(t, c2)

	payload := []byte(`{"v":1,"t":"relay"}`)
	h.broadcastLocal("proj-1", payload, c1)

	select {
	case got := <-c2.send:
		assert.Equal(t, payload, got)
	default:
		t.Fatal("expected c2 to receive broadcast")
	}

	select {
	case <-c1.send:
		t.Fatal("sender should be excluded")
	default:
	}

	drainConnSend(t, c1)
	drainConnSend(t, c2)
	h.unregister(c1)
	drainConnSend(t, c2)
	h.unregister(c2)

	h.mu.RLock()
	_, exists := h.rooms["proj-1"]
	h.mu.RUnlock()
	assert.False(t, exists, "room should be removed when empty")
}

func TestPresenceBroadcast_photoURL(t *testing.T) {
	h := NewHub(Options{})
	c1 := &Conn{
		hub: h, projectID: "proj-1", userID: "alice", send: make(chan []byte, 16),
		bgCtx: context.Background(),
	}
	c2 := &Conn{
		hub: h, projectID: "proj-1", userID: "bob", photoURL: "https://cdn.example/b.png",
		send: make(chan []byte, 16), bgCtx: context.Background(),
	}
	h.register(c1)
	drainConnSend(t, c1)
	h.register(c2)
	drainConnSend(t, c2)

	select {
	case got := <-c1.send:
		var env struct {
			T string         `json:"t"`
			D map[string]any `json:"d"`
		}
		require.NoError(t, json.Unmarshal(got, &env))
		assert.Equal(t, "presence", env.T)
		assert.Equal(t, "join", env.D["event"])
		assert.Equal(t, "bob", env.D["userId"])
		assert.Equal(t, "https://cdn.example/b.png", env.D["photoURL"])
	default:
		t.Fatal("expected c1 to receive presence join for bob")
	}
}
