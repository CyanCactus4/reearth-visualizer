package collab

import "context"

// ApplyAuditRecord is one persisted successful collab apply (PLAN phase 6 journal slice).
type ApplyAuditRecord struct {
	ProjectID string
	UserID    string
	Kind      string
	SceneRev  int64
	SceneID   string
	WidgetID  string
}

// ApplyAuditStore appends successful apply operations for auditing / future undo UI.
type ApplyAuditStore interface {
	Append(ctx context.Context, rec ApplyAuditRecord) error
}
