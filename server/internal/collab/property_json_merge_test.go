package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONMergePatchObject_shallow(t *testing.T) {
	dst := map[string]any{
		"a": map[string]any{"type": "NUMBER", "value": 1.0},
		"b": map[string]any{"type": "STRING", "value": "x"},
	}
	patch := map[string]any{
		"a": map[string]any{"type": "NUMBER", "value": 2.0},
	}
	out := jsonMergePatchObject(dst, patch)
	assert.Equal(t, 2.0, out["a"].(map[string]any)["value"])
	assert.Equal(t, "x", out["b"].(map[string]any)["value"])
}

func TestJSONMergePatchObject_rejectsImplicitDelete(t *testing.T) {
	orig := map[string]any{"k": map[string]any{"type": "NUMBER", "value": 1.0}}
	patch := map[string]any{"k": nil}
	out := jsonMergePatchObject(orig, patch)
	_, ok := out["k"]
	assert.False(t, ok)
}
