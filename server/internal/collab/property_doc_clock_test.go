package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPropertyDocClock_inMemory(t *testing.T) {
	h := NewHub(Options{})
	sid := "sc1"
	pid := "pr1"
	assert.Equal(t, int64(0), h.PropertyDocClock(sid, pid))
	n := h.BumpPropertyDocClock(sid, pid)
	require.Equal(t, int64(1), n)
	assert.Equal(t, int64(1), h.PropertyDocClock(sid, pid))
}
