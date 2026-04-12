package collab

import (
	"encoding/json"
	"testing"

	accountsID "github.com/reearth/reearth-accounts/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRemoveWidgetInverseJSON(t *testing.T) {
	raw := buildRemoveWidgetInverseJSON("sc1", "builtin", "wid1")
	require.NotEmpty(t, raw)
	var inv applyRemoveWidget
	require.NoError(t, json.Unmarshal(raw, &inv))
	assert.Equal(t, "remove_widget", inv.Kind)
	assert.Equal(t, "sc1", inv.SceneID)
	assert.Equal(t, "wid1", inv.WidgetID)
}

func TestBuildAddWidgetInverseJSON(t *testing.T) {
	wid := id.NewWidgetID()
	pid := id.OfficialPluginID
	w := scene.MustWidget(wid, pid, "ext1", id.NewPropertyID(), true, false)
	ws := scene.NewWidgets([]*scene.Widget{w}, nil)
	sid := id.NewSceneID()
	sc := scene.New().ID(sid).Workspace(accountsID.NewWorkspaceID()).Widgets(ws).MustBuild()

	raw := buildAddWidgetInverseJSON(sc, sc.ID().String(), "builtin", wid)
	require.NotEmpty(t, raw)
	var inv applyAddWidget
	require.NoError(t, json.Unmarshal(raw, &inv))
	assert.Equal(t, "add_widget", inv.Kind)
	assert.Equal(t, pid.String(), inv.PluginID)
	assert.Equal(t, "ext1", inv.ExtensionID)
}
