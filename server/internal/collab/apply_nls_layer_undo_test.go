package collab

import (
	"encoding/json"
	"testing"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/nlslayer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUpdateNLSLayerInverseJSON_name(t *testing.T) {
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	lid := id.NewNLSLayerID()
	cfg := nlslayer.Config{"a": float64(1)}
	layer := nlslayer.NewNLSLayerSimple().ID(lid).Scene(sid).Title("OldTitle").LayerType(nlslayer.Simple).Config(&cfg).MustBuild()
	newName := "NewTitle"
	fwd := &applyUpdateNLSLayer{
		Kind:    "update_nls_layer",
		SceneID: sid.String(),
		LayerID: lid.String(),
		Name:    &newName,
	}
	raw := buildUpdateNLSLayerInverseJSON(layer, fwd, true, false, false, false)
	require.NotNil(t, raw)
	var inv applyUpdateNLSLayer
	require.NoError(t, json.Unmarshal(raw, &inv))
	require.NotNil(t, inv.Name)
	assert.Equal(t, "OldTitle", *inv.Name)
}

func TestBuildUpdateNLSLayerInverseJSON_configWhenOldNil(t *testing.T) {
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	lid := id.NewNLSLayerID()
	layer := nlslayer.NewNLSLayerSimple().ID(lid).Scene(sid).Title("T").LayerType(nlslayer.Simple).MustBuild()
	fwd := &applyUpdateNLSLayer{
		Kind:    "update_nls_layer",
		SceneID: sid.String(),
		LayerID: lid.String(),
		Config:  json.RawMessage(`{"x":1}`),
	}
	assert.Nil(t, buildUpdateNLSLayerInverseJSON(layer, fwd, false, false, false, true))
}

func TestReverseUpdateNLSLayerItems(t *testing.T) {
	a := applyUpdateNLSLayerItem{LayerID: "a"}
	b := applyUpdateNLSLayerItem{LayerID: "b"}
	c := applyUpdateNLSLayerItem{LayerID: "c"}
	s := []applyUpdateNLSLayerItem{a, b, c}
	reverseUpdateNLSLayerItems(s)
	assert.Equal(t, "c", s[0].LayerID)
	assert.Equal(t, "b", s[1].LayerID)
	assert.Equal(t, "a", s[2].LayerID)
}

func TestBuildUpdateNLSLayerInverseJSON_indexFromUnset(t *testing.T) {
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	lid := id.NewNLSLayerID()
	layer := nlslayer.NewNLSLayerSimple().ID(lid).Scene(sid).Title("T").LayerType(nlslayer.Simple).MustBuild()
	newIdx := 2
	fwd := &applyUpdateNLSLayer{
		Kind:    "update_nls_layer",
		SceneID: sid.String(),
		LayerID: lid.String(),
		Index:   &newIdx,
	}
	assert.Nil(t, buildUpdateNLSLayerInverseJSON(layer, fwd, false, false, true, false))
}
