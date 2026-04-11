package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidWidgetAreaLockID(t *testing.T) {
	assert.True(t, validWidgetAreaLockID("inner:center:middle"))
	assert.False(t, validWidgetAreaLockID("invalid"))
	assert.False(t, validWidgetAreaLockID("inner:center"))
	assert.False(t, validWidgetAreaLockID("inner:left:invalid"))
}
