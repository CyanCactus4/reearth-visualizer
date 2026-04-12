package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inverse merge for undo uses the same jsonMergePatchObject as forward merge (flat leaf maps).
func TestMergePropertyJSON_inversePatchRestoresLeaves(t *testing.T) {
	sep := "\x00"
	k := "tiles" + sep + "group1" + sep + "visible"
	origGen := map[string]any{
		k: map[string]any{"type": "BOOL", "value": true},
	}
	patch := map[string]any{
		k: map[string]any{"type": "BOOL", "value": false},
	}
	merged := jsonMergePatchObject(origGen, patch)
	require.False(t, merged[k].(map[string]any)["value"].(bool))

	inverse := map[string]any{
		k: cloneJSONMapStringAny(toStringAnyMap(origGen[k])),
	}
	restored := jsonMergePatchObject(merged, inverse)
	assert.Equal(t, true, restored[k].(map[string]any)["value"].(bool))
}
