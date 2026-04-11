package collab

import (
	"testing"

	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAlignSystem(t *testing.T) {
	a, err := parseAlignSystem("desktop")
	require.NoError(t, err)
	assert.Equal(t, scene.WidgetAlignSystemTypeDesktop, a)

	_, err = parseAlignSystem("tablet")
	assert.Error(t, err)
}

func TestWidgetLocationValid(t *testing.T) {
	assert.True(t, widgetLocationValid(scene.WidgetLocation{
		Zone:    scene.WidgetZoneOuter,
		Section: scene.WidgetSectionCenter,
		Area:    scene.WidgetAreaTop,
	}))
	assert.False(t, widgetLocationValid(scene.WidgetLocation{
		Zone:    "x",
		Section: scene.WidgetSectionCenter,
		Area:    scene.WidgetAreaTop,
	}))
}
