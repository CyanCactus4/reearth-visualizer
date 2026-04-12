package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockHolderWire_Parse(t *testing.T) {
	assert.Equal(t, "u1", LockHolderWire("u1", ""))
	hu, hc := ParseLockHolderWire("u1")
	assert.Equal(t, "u1", hu)
	assert.Equal(t, "", hc)

	w := LockHolderWire("u1", "c9")
	assert.Contains(t, w, "u1")
	hu2, hc2 := ParseLockHolderWire(w)
	assert.Equal(t, "u1", hu2)
	assert.Equal(t, "c9", hc2)
}

func TestLockHeldBySameTab(t *testing.T) {
	w := LockHolderWire("alice", "t1")
	assert.True(t, LockHeldBySameTab(w, "alice", "t1"))
	assert.False(t, LockHeldBySameTab(w, "alice", "t2"))
	assert.False(t, LockHeldBySameTab(w, "bob", "t1"))
	assert.True(t, LockHeldBySameTab("alice", "alice", ""))
}

func TestHTTPLockBlocksUser(t *testing.T) {
	assert.False(t, HTTPLockBlocksUser("", "alice"))
	assert.True(t, HTTPLockBlocksUser("bob", "alice"))
	assert.False(t, HTTPLockBlocksUser("alice", "alice"))
	assert.True(t, HTTPLockBlocksUser(LockHolderWire("alice", "t1"), "alice"))
}

func TestNormalizeCollabClientID(t *testing.T) {
	assert.Equal(t, "", NormalizeCollabClientID(""))
	assert.Equal(t, "ab-12_CD", NormalizeCollabClientID(" ab-12_CD "))
	assert.Equal(t, "", NormalizeCollabClientID("bad id"))
	assert.Equal(t, "", NormalizeCollabClientID("a/../b"))
}
