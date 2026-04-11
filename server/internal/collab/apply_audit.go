package collab

import "context"

// ApplyAuditRecord is one persisted successful collab apply (PLAN phase 6 journal slice).
type ApplyAuditRecord struct {
	ProjectID  string
	UserID     string
	Kind       string
	SceneRev   int64
	SceneID    string
	WidgetID   string
	StoryID    string
	PageID     string
	BlockID    string
	PropertyID string
	FieldID    string
	StyleID    string
}

// ApplyAuditListRow is one row returned by GET /api/collab/apply-audit (newest first).
type ApplyAuditListRow struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Kind       string `json:"kind"`
	SceneRev   int64  `json:"sceneRev"`
	SceneID    string `json:"sceneId,omitempty"`
	WidgetID   string `json:"widgetId,omitempty"`
	StoryID    string `json:"storyId,omitempty"`
	PageID     string `json:"pageId,omitempty"`
	BlockID    string `json:"blockId,omitempty"`
	PropertyID string `json:"propertyId,omitempty"`
	FieldID    string `json:"fieldId,omitempty"`
	StyleID    string `json:"styleId,omitempty"`
	Ts         int64  `json:"ts"`
}

// ApplyAuditStore appends successful apply operations for auditing / future undo UI.
type ApplyAuditStore interface {
	Append(ctx context.Context, rec ApplyAuditRecord) error
	ListRecent(ctx context.Context, projectID string, limit int) ([]ApplyAuditListRow, error)
}
