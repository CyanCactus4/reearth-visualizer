package collab

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeApplyUpdatePropertyValue_stringField(t *testing.T) {
	raw := []byte(`{
		"kind":"update_property_value",
		"sceneId":"01fbpdqax0ttrftj3gb5gm4rw7",
		"propertyId":"01fbpdqax0ttrftj3gb5gm4rw8",
		"fieldId":"title",
		"type":"STRING",
		"value":"hello"
	}`)
	p, ptr, val, err := decodeApplyUpdatePropertyValue(raw)
	require.NoError(t, err)
	assert.Equal(t, "title", p.FieldID)
	assert.Equal(t, "STRING", p.Type)
	require.NotNil(t, ptr)
	require.NotNil(t, val)
	s := val.ValueString()
	require.NotNil(t, s)
	assert.Equal(t, "hello", *s)
}

func TestDecodeApplyUpdatePropertyValue_clearValue(t *testing.T) {
	raw := []byte(`{
		"kind":"update_property_value",
		"sceneId":"01fbpdqax0ttrftj3gb5gm4rw7",
		"propertyId":"01fbpdqax0ttrftj3gb5gm4rw8",
		"fieldId":"x",
		"type":"STRING"
	}`)
	p, ptr, val, err := decodeApplyUpdatePropertyValue(raw)
	require.NoError(t, err)
	assert.Equal(t, "x", p.FieldID)
	require.NotNil(t, ptr)
	assert.Nil(t, val)
}
