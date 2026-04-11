package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHub_NoRedis(t *testing.T) {
	h := NewHub("")
	require.NotNil(t, h)
	assert.Nil(t, h.relay)
}

func TestHubRegisterBroadcastUnregister(t *testing.T) {
	h := NewHub("")
	c1 := &Conn{hub: h, projectID: "proj-1", send: make(chan []byte, 16)}
	c2 := &Conn{hub: h, projectID: "proj-1", send: make(chan []byte, 16)}

	h.register(c1)
	h.register(c2)

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

	h.unregister(c1)
	h.unregister(c2)

	h.mu.RLock()
	_, exists := h.rooms["proj-1"]
	h.mu.RUnlock()
	assert.False(t, exists, "room should be removed when empty")
}
