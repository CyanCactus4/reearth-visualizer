package collab

import (
	"encoding/json"
	"testing"

	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/storytelling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildUpdateStoryPageInverseJSON_title(t *testing.T) {
	sid := id.MustSceneID("01fbpdqax0ttrftj3gb5gm4rw7")
	pageID := id.NewPageID()
	pg := storytelling.NewPage().ID(pageID).Title("Old").MustBuild()
	st := storytelling.NewStory().NewID().
		Scene(sid).
		Property(id.NewPropertyID()).
		Project(id.NewProjectID()).
		Pages(storytelling.NewPageList([]*storytelling.Page{pg})).
		MustBuild()

	newTitle := "New"
	fwd := &applyUpdateStoryPage{
		Kind:    "update_story_page",
		SceneID: sid.String(),
		StoryID: st.Id().String(),
		PageID:  pageID.String(),
		Title:   &newTitle,
	}
	raw := buildUpdateStoryPageInverseJSON(st, pg, fwd)
	require.NotNil(t, raw)
	var inv applyUpdateStoryPage
	require.NoError(t, json.Unmarshal(raw, &inv))
	require.NotNil(t, inv.Title)
	assert.Equal(t, "Old", *inv.Title)
}
