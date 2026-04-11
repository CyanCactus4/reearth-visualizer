package collab

import (
	"context"
	"encoding/json"

	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
)

// baseSceneRevFromApplyPayload returns client-observed scene revision (scene.UpdatedAt ms) if present.
func baseSceneRevFromApplyPayload(d json.RawMessage) (int64, bool) {
	var x struct {
		BaseSceneRev *int64 `json:"baseSceneRev"`
	}
	if err := json.Unmarshal(d, &x); err != nil || x.BaseSceneRev == nil {
		return 0, false
	}
	return *x.BaseSceneRev, true
}

// assertSceneRevIfPresent rejects the apply when baseSceneRev does not match current scene.UpdatedAt (coarse OT guard).
func assertSceneRevIfPresent(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, from *Conn, d json.RawMessage) bool {
	br, ok := baseSceneRevFromApplyPayload(d)
	if !ok {
		return true
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil || len(scenes) == 0 {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": "scene fetch failed"}})
		return false
	}
	cur := scenes[0].UpdatedAt().UnixMilli()
	if cur != br {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
			"code":    "stale_state",
			"message": "scene changed since baseSceneRev; refetch and retry",
		}})
		return false
	}
	return true
}
