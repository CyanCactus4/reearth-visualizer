package collab

import (
	"encoding/json"
	"testing"

	accountsID "github.com/reearth/reearth-accounts/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/stretchr/testify/require"
)

func TestPatchedRedoRemoveWidgetForward(t *testing.T) {
	pid := id.OfficialPluginID
	ext := id.PluginExtensionID("ext1")
	wLow := scene.MustWidget(id.MustWidgetID("01fbpdqax0ttrftj3gb5gm4wg0"), pid, ext, id.NewPropertyID(), true, false)
	wHigh := scene.MustWidget(id.MustWidgetID("01fbpdqax0ttrftj3gb5gm4wg9"), pid, ext, id.NewPropertyID(), true, false)
	ws := scene.NewWidgets([]*scene.Widget{wLow, wHigh}, nil)
	sid := id.NewSceneID()
	sc := scene.New().ID(sid).Workspace(accountsID.NewWorkspaceID()).Widgets(ws).MustBuild()

	rec := &UndoableOpRecord{
		Kind: "remove_widget",
		Forward: mustJSON(t, applyRemoveWidget{
			Kind: "remove_widget", SceneID: sid.String(), AlignSystem: "builtin",
			WidgetID: wLow.ID().String(),
		}),
		Inverse: mustJSON(t, applyAddWidget{
			Kind: "add_widget", SceneID: sid.String(), AlignSystem: "builtin",
			PluginID: pid.String(), ExtensionID: string(ext),
		}),
	}
	out, err := patchedRedoRemoveWidgetForward(rec, sc)
	require.NoError(t, err)
	var fwd applyRemoveWidget
	require.NoError(t, json.Unmarshal(out, &fwd))
	require.Equal(t, wHigh.ID().String(), fwd.WidgetID)
}

func mustJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return json.RawMessage(b)
}
