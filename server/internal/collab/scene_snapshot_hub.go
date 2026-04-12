package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/reearth/reearth/server/internal/adapter"
	"github.com/reearth/reearth/server/pkg/id"
	"github.com/reearth/reearth/server/pkg/scene"
	"github.com/reearth/reearthx/log"
)

const defaultSceneSnapshotMinInterval = 45 * time.Second

// queueSceneSnapshot asynchronously persists a sparse ExportSceneData snapshot (rate-limited per scene).
func (h *Hub) queueSceneSnapshot(from *Conn, sc *scene.Scene, rev int64) {
	if h == nil || h.sceneSnapshotStore == nil || from == nil || sc == nil || rev <= 0 {
		return
	}
	sid := sc.ID().String()
	h.snapMu.Lock()
	last := h.snapLastAt[sid]
	if !last.IsZero() && time.Since(last) < defaultSceneSnapshotMinInterval {
		h.snapMu.Unlock()
		return
	}
	h.snapLastAt[sid] = time.Now()
	h.snapMu.Unlock()

	go h.captureSceneSnapshot(from, sc, rev)
}

func (h *Hub) captureSceneSnapshot(from *Conn, sc *scene.Scene, rev int64) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(from.bgCtx), 60*time.Second)
	defer cancel()
	uc := adapter.Usecases(from.bgCtx)
	if uc == nil || from.operator == nil {
		return
	}
	pid, err := id.ProjectIDFrom(from.projectID)
	if err != nil {
		return
	}
	prj, err := uc.Project.FindActiveById(ctx, pid, from.operator)
	if err != nil || prj == nil {
		return
	}
	_, payload, err := uc.Scene.ExportSceneData(ctx, prj)
	if err != nil {
		log.Warnfc(ctx, "collab: scene snapshot export: %v", err)
		return
	}
	b, err := json.Marshal(payload)
	if err != nil || len(b) > 12<<20 {
		return
	}
	rec := SceneSnapshotRecord{
		ProjectID: from.projectID,
		SceneID:   sc.ID().String(),
		SceneRev:  rev,
		Data:      b,
		Ts:        time.Now().UnixMilli(),
	}
	if err := h.sceneSnapshotStore.Append(ctx, rec); err != nil {
		log.Warnfc(ctx, "collab: scene snapshot append: %v", err)
	}
}
