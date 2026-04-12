package collab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/reearth/reearth/server/internal/usecase"
	"github.com/reearth/reearth/server/internal/usecase/interfaces"
	"github.com/reearth/reearth/server/pkg/id"
)

var errStaleSceneRev = errors.New("stale_scene_rev")

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

// assertSceneRevIfPresentErr returns errStaleSceneRev when baseSceneRev is set and does not match scene.UpdatedAt.
func assertSceneRevIfPresentErr(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, d json.RawMessage) error {
	br, ok := baseSceneRevFromApplyPayload(d)
	if !ok {
		return nil
	}
	opCtx, cancel := context.WithTimeout(ctx, applyOpTimeout)
	defer cancel()
	scenes, err := uc.Scene.Fetch(opCtx, []id.SceneID{sid}, op)
	if err != nil {
		return fmt.Errorf("scene fetch failed: %w", err)
	}
	if len(scenes) == 0 {
		return errors.New("scene fetch failed")
	}
	cur := scenes[0].UpdatedAt().UnixMilli()
	if cur != br {
		return errStaleSceneRev
	}
	return nil
}

// assertSceneRevIfPresent rejects the apply when baseSceneRev does not match current scene.UpdatedAt (coarse OT guard).
func assertSceneRevIfPresent(ctx context.Context, uc *interfaces.Container, op *usecase.Operator, sid id.SceneID, from *Conn, d json.RawMessage) bool {
	err := assertSceneRevIfPresentErr(ctx, uc, op, sid, d)
	if err == nil {
		return true
	}
	if errors.Is(err, errStaleSceneRev) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{
			"code":    "stale_state",
			"message": "scene changed since baseSceneRev; refetch and retry",
		}})
		return false
	}
	from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "internal", "message": err.Error()}})
	return false
}
