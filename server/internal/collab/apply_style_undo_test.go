package collab

import (
	"encoding/json"
	"testing"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUpdateStyleInverseJSON_nameAndValue(t *testing.T) {
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	st := scene.NewStyle().NewID().Scene(sid).Name("OldName").Value(&scene.StyleValue{"k": "v"}).MustBuild()

	fwd := applyUpdateStyle{
		Kind:    "update_style",
		SceneID: "01fbpdqax0ttrftj3gb5gm4rw7",
		StyleID: "01fbpdqax0ttrftj3gb5gm4rw9",
	}
	invRaw := buildUpdateStyleInverseJSON(st, &fwd, true, true)
	require.NotNil(t, invRaw)
	var inv applyUpdateStyle
	require.NoError(t, json.Unmarshal(invRaw, &inv))
	assert.Equal(t, "OldName", *inv.Name)
	require.NotEmpty(t, inv.Value)
}

func TestBuildUpdateStyleInverseJSON_valueOnlyOldNil(t *testing.T) {
	st := scene.NewStyle().NewID().MustBuild()
	fwd := applyUpdateStyle{Kind: "update_style", SceneID: "s", StyleID: "y"}
	assert.Nil(t, buildUpdateStyleInverseJSON(st, &fwd, false, true))
}
