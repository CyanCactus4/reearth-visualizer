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

func TestApplyMoveStoryPageInverseJSON_shape(t *testing.T) {
	inv := applyMoveStoryPage{
		Kind:    "move_story_page",
		SceneID: "01fbpdqax0ttrftj3gb5gm4rw7",
		StoryID: "01fbpdqax0ttrftj3gb5gm4rw8",
		PageID:  "01fbpdqax0ttrftj3gb5gm4rw9",
		Index:   2,
	}
	b, err := json.Marshal(inv)
	require.NoError(t, err)
	var out applyMoveStoryPage
	require.NoError(t, json.Unmarshal(b, &out))
	assert.Equal(t, 2, out.Index)
	assert.Equal(t, "move_story_page", out.Kind)
}
