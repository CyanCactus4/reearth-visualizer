package collab

import (
	"context"
	"encoding/json"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
)

type applyMoveStoryBlock struct {
	Kind        string `json:"kind"`
	SceneID     string `json:"sceneId"`
	StoryID     string `json:"storyId"`
	PageID      string `json:"pageId"`
	BlockID     string `json:"blockId"`
	Index       int    `json:"index"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func applyMoveStoryBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyMoveStoryBlock
	if err := json.Unmarshal(d, &p); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_payload", "message": err.Error()}})
		return nil
	}
	op := from.operator
	if op == nil || !op.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "write not allowed"}})
		return nil
	}
	sid, err := id.SceneIDFrom(p.SceneID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_scene", "message": err.Error()}})
		return nil
	}
	if sid != from.sceneID {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "scene_mismatch", "message": "scene does not belong to this room"}})
		return nil
	}

	uc := adapter.Usecases(ctx)
	if uc == nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "usecases unavailable"}})
		return nil
	}
	if !assertSceneRevIfPresent(ctx, uc, op, sid, from, d) {
		return nil
	}

	storyID, err := id.StoryIDFrom(p.StoryID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_story", "message": err.Error()}})
		return nil
	}
	pageID, err := id.PageIDFrom(p.PageID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_page", "message": err.Error()}})
		return nil
	}
	blockID, err := id.BlockIDFrom(p.BlockID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_block", "message": err.Error()}})
		return nil
	}

	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()

	_, _, _, _, err2 := uc.StoryTelling.MoveBlock(opCtx, interfaces.MoveBlockParam{
		StoryID: storyID,
		PageID:  pageID,
		BlockID: blockID,
		Index:   p.Index,
	}, op)
	if err2 != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "apply_failed", "message": err2.Error()}})
		return nil
	}

	scenes, err3 := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err3 != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene reload failed"}})
		return nil
	}
	sc := scenes[0]
	broadcastApplied(ctx, hub, from, "move_story_block", map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
		"blockId": p.BlockID,
	}, sc)
	return nil
}
