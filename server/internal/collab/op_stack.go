package collab

import (
	"context"
	"encoding/json"
)

// UndoableOpRecord is one undoable collab operation owned by a single user on a scene.
type UndoableOpRecord struct {
	ProjectID string
	SceneID   string
	UserID    string
	Kind      string
	Forward   json.RawMessage
	Inverse   json.RawMessage
}

// CollabOpStack persists per-user undo/redo stacks for a scene (Mongo when configured).
type CollabOpStack interface {
	RecordUndoable(ctx context.Context, rec UndoableOpRecord) error
	Undo(ctx context.Context, userID, sceneID string) (*UndoableOpRecord, error)
	Redo(ctx context.Context, userID, sceneID string) (*UndoableOpRecord, error)
	// PatchHeadRedoForward rewrites forwardJson for the op at the tail of the redo stack (Mongo).
	// In-memory / no-op implementations should return nil.
	PatchHeadRedoForward(ctx context.Context, userID, sceneID string, forward json.RawMessage) error
}
