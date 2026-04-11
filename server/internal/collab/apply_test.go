package collab

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDispatchApply_unknownKind(t *testing.T) {
	h := NewHub(Options{})
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	c := &Conn{
		hub:       h,
		projectID: "p1",
		sceneID:   sid,
		userID:    "alice",
		send:      make(chan []byte, 8),
		bgCtx:     context.Background(),
	}
	err := dispatchApply(context.Background(), h, c, []byte(`{"kind":"no_such_op"}`))
	require.NoError(t, err)
	select {
	case b := <-c.send:
		var sm serverMessage
		require.NoError(t, json.Unmarshal(b, &sm))
		assert.Equal(t, "error", sm.T)
	default:
		t.Fatal("expected error message")
	}
}

func TestDispatchApply_invalidJSON(t *testing.T) {
	h := NewHub(Options{})
	c := &Conn{hub: h, projectID: "p1", send: make(chan []byte, 8), bgCtx: context.Background()}
	err := dispatchApply(context.Background(), h, c, []byte(`{`))
	require.NoError(t, err)
	select {
	case b := <-c.send:
		var sm serverMessage
		require.NoError(t, json.Unmarshal(b, &sm))
		assert.Equal(t, "error", sm.T)
	default:
		t.Fatal("expected error message")
	}
}

func TestLockTable_lookup(t *testing.T) {
	lt := newLockTable()
	ok, holder, _ := lt.TryAcquire("p", "widget", "w1", "u1", 5*time.Minute)
	assert.True(t, ok)
	assert.Equal(t, "u1", holder)
	h, active := lt.Lookup("p", "widget", "w1")
	assert.True(t, active)
	assert.Equal(t, "u1", h)
	_, active2 := lt.Lookup("p", "widget", "missing")
	assert.False(t, active2)
}
