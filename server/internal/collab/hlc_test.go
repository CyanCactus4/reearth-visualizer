package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHLCCompare_totalOrder(t *testing.T) {
	a := HLC{Physical: 2, Logical: 0, NodeID: "a"}
	b := HLC{Physical: 1, Logical: 99, NodeID: "z"}
	assert.True(t, a.After(b))

	x := HLC{Physical: 5, Logical: 1, NodeID: "m1"}
	y := HLC{Physical: 5, Logical: 1, NodeID: "m2"}
	assert.True(t, y.After(x))
}

func TestHLCTick_monotonic(t *testing.T) {
	h := HLC{Physical: 10, Logical: 3, NodeID: "n"}
	n1 := h.Tick(10)
	assert.Equal(t, int64(10), n1.Physical)
	assert.Equal(t, uint32(4), n1.Logical)

	n2 := n1.Tick(20)
	assert.Equal(t, int64(20), n2.Physical)
	assert.Equal(t, uint32(0), n2.Logical)
}

func TestHLCReceive_converges(t *testing.T) {
	var local HLC
	local.Physical = 5
	local.Logical = 2
	local.NodeID = "a"
	remote := HLC{Physical: 5, Logical: 4, NodeID: "b"}
	local.Receive(5, remote)
	assert.True(t, local.Logical >= 5)
}
