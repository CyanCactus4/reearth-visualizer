package collab

import "context"

// SceneSnapshotRecord is a point-in-time export payload for admin restore (TASK FR-5).
type SceneSnapshotRecord struct {
	ProjectID string
	SceneID   string
	SceneRev  int64
	Data      []byte
	Ts        int64
}

// SceneSnapshotStore persists sparse scene JSON exports keyed by scene revision.
type SceneSnapshotStore interface {
	Append(ctx context.Context, rec SceneSnapshotRecord) error
	// LoadClosestAtOrBelow returns the newest snapshot with sceneRev <= targetRev (newest-first).
	LoadClosestAtOrBelow(ctx context.Context, sceneID string, targetRev int64) (data []byte, sceneRev int64, err error)
}
