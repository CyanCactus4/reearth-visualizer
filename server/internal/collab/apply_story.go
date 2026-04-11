package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearthx/log"
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

	fromIdx := -1
	if sl, e := uc.StoryTelling.Fetch(opCtx, id.StoryIDList{storyID}, op); e == nil && sl != nil && len(*sl) > 0 {
		for _, st := range *sl {
			if st.Id() != storyID {
				continue
			}
			pg := st.Pages().Page(pageID)
			if pg == nil {
				break
			}
			for i, b := range pg.Blocks() {
				if b.ID() == blockID {
					fromIdx = i
					break
				}
			}
			break
		}
	}

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

	if hub != nil && hub.opStack != nil && fromIdx >= 0 {
		inv := applyMoveStoryBlock{
			Kind:    "move_story_block",
			SceneID: p.SceneID,
			StoryID: p.StoryID,
			PageID:  p.PageID,
			BlockID: p.BlockID,
			Index:   fromIdx,
		}
		binv, errI := json.Marshal(inv)
		if errI == nil {
			rec := UndoableOpRecord{
				ProjectID: from.projectID,
				SceneID:   sid.String(),
				UserID:    actorUserID(from),
				Kind:      "move_story_block",
				Forward:   d,
				Inverse:   json.RawMessage(binv),
			}
			go func() {
				pctx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel2()
				if err := hub.opStack.RecordUndoable(pctx, rec); err != nil {
					log.Warnfc(pctx, "collab: undo stack: %v", err)
				}
			}()
		}
	}
	return nil
}

type applyCreateStoryBlock struct {
	Kind          string `json:"kind"`
	SceneID       string `json:"sceneId"`
	StoryID       string `json:"storyId"`
	PageID        string `json:"pageId"`
	PluginID      string `json:"pluginId"`
	ExtensionID   string `json:"extensionId"`
	Index         *int   `json:"index,omitempty"`
	BaseSceneRev  *int64 `json:"baseSceneRev,omitempty"`
}

type applyRemoveStoryBlock struct {
	Kind         string `json:"kind"`
	SceneID      string `json:"sceneId"`
	StoryID      string `json:"storyId"`
	PageID       string `json:"pageId"`
	BlockID      string `json:"blockId"`
	BaseSceneRev *int64 `json:"baseSceneRev,omitempty"`
}

func applyCreateStoryBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyCreateStoryBlock
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
	pid, err := id.PluginIDFrom(p.PluginID)
	if err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_plugin", "message": err.Error()}})
		return nil
	}
	eid := id.PluginExtensionID(p.ExtensionID)
	if string(eid) == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_extension", "message": "extensionId required"}})
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	_, _, block, _, err2 := uc.StoryTelling.CreateBlock(opCtx, interfaces.CreateBlockParam{
		StoryID:     storyID,
		PageID:      pageID,
		PluginID:    pid,
		ExtensionID: eid,
		Index:       p.Index,
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
	extra := map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
	}
	if block != nil {
		extra["blockId"] = block.ID().String()
	}
	broadcastApplied(ctx, hub, from, "create_story_block", extra, sc)
	return nil
}

func applyRemoveStoryBlockOp(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var p applyRemoveStoryBlock
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
	_, _, _, err2 := uc.StoryTelling.RemoveBlock(opCtx, interfaces.RemoveBlockParam{
		StoryID: storyID,
		PageID:  pageID,
		BlockID: blockID,
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
	broadcastApplied(ctx, hub, from, "remove_story_block", map[string]any{
		"sceneId": p.SceneID,
		"storyId": p.StoryID,
		"pageId":  p.PageID,
		"blockId": p.BlockID,
	}, sc)
	return nil
}
