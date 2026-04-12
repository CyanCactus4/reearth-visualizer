package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyFieldClock_inMemory(t *testing.T) {
	h := NewHub(Options{})
	sid := "sc1"
	pid := "pr1"
	n := h.BumpPropertyFieldClock(sid, pid, "g1", "it1", "f1")
	assert.Equal(t, int64(1), n)
	assert.Equal(t, int64(1), h.PropertyFieldClock(sid, pid, "g1", "it1", "f1"))
	n2 := h.BumpPropertyFieldClock(sid, pid, "g1", "it1", "f1")
	require.Equal(t, int64(2), n2)
}

func TestPropertyFieldClock_independentPerFieldKey(t *testing.T) {
	h := NewHub(Options{})
	sid := "sc1"
	pid := "pr1"
	h.BumpPropertyFieldClock(sid, pid, "g1", "it1", "opacity")
	h.BumpPropertyFieldClock(sid, pid, "g1", "it1", "color")
	assert.Equal(t, int64(1), h.PropertyFieldClock(sid, pid, "g1", "it1", "opacity"))
	assert.Equal(t, int64(1), h.PropertyFieldClock(sid, pid, "g1", "it1", "color"))
	assert.Equal(t, int64(0), h.PropertyFieldClock(sid, pid, "g1", "it1", "other"))
}
