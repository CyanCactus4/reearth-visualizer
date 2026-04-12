package collab

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLockTableAcquireRelease(t *testing.T) {
	tb := newLockTable()
	ttl := time.Minute

	ok, holder, until := tb.TryAcquire("p1", "layer", "lid1", "alice", "", ttl)
	require.True(t, ok)
	assert.Equal(t, "alice", holder)
	assert.True(t, until.After(time.Now()))

	ok2, holder2, _ := tb.TryAcquire("p1", "layer", "lid1", "bob", "", ttl)
	assert.False(t, ok2)
	assert.Equal(t, "alice", holder2)

	assert.True(t, tb.Release("p1", "layer", "lid1", "alice", ""))

	ok3, holder3, _ := tb.TryAcquire("p1", "layer", "lid1", "bob", "", ttl)
	assert.True(t, ok3)
	assert.Equal(t, "bob", holder3)
}

func TestLockTableHeartbeatWrongUser(t *testing.T) {
	tb := newLockTable()
	ttl := time.Minute
	ok, _, _ := tb.TryAcquire("p", "widget", "w1", "alice", "", ttl)
	require.True(t, ok)
	okHB, _ := tb.Heartbeat("p", "widget", "w1", "bob", "", ttl)
	assert.False(t, okHB)
	okHB2, until2 := tb.Heartbeat("p", "widget", "w1", "alice", "", ttl)
	assert.True(t, okHB2)
	assert.True(t, until2.After(time.Now()))
}

func TestLockTableSameUserTwoTabs(t *testing.T) {
	tb := newLockTable()
	ttl := time.Minute
	ok, w1, _ := tb.TryAcquire("p", "layer", "L1", "alice", "tab1", ttl)
	require.True(t, ok)
	assert.Equal(t, LockHolderWire("alice", "tab1"), w1)
	ok2, w2, _ := tb.TryAcquire("p", "layer", "L1", "alice", "tab2", ttl)
	assert.False(t, ok2)
	assert.Equal(t, w1, w2)
}
