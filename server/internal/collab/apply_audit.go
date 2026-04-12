package collab

import "context"

// ApplyAuditRecord is one persisted successful collab apply (PLAN phase 6 journal slice).
type ApplyAuditRecord struct {
	ProjectID string
	UserID    string
	UserName  string
	Kind      string
	// OpKind is the domain apply kind from the undo stack (e.g. update_widget) when Kind is collab_undo / collab_redo.
	OpKind     string
	SceneRev   int64
	SceneID    string
	WidgetID   string
	StoryID    string
	PageID     string
	BlockID    string
	PropertyID string
	FieldID    string
	StyleID    string
	LayerID    string
	LayerIDs   []string
}

// ApplyAuditListRow is one row returned by GET /api/collab/apply-audit (newest first).
type ApplyAuditListRow struct {
	ID         string   `json:"id"`
	UserID     string   `json:"userId"`
	UserName   string   `json:"userName,omitempty"`
	Kind       string   `json:"kind"`
	OpKind     string   `json:"opKind,omitempty"`
	SceneRev   int64    `json:"sceneRev"`
	SceneID    string   `json:"sceneId,omitempty"`
	WidgetID   string   `json:"widgetId,omitempty"`
	StoryID    string   `json:"storyId,omitempty"`
	PageID     string   `json:"pageId,omitempty"`
	BlockID    string   `json:"blockId,omitempty"`
	PropertyID string   `json:"propertyId,omitempty"`
	FieldID    string   `json:"fieldId,omitempty"`
	StyleID    string   `json:"styleId,omitempty"`
	LayerID    string   `json:"layerId,omitempty"`
	LayerIDs   []string `json:"layerIds,omitempty"`
	Ts         int64    `json:"ts"`
}

// ApplyAuditStore appends successful apply operations for auditing / future undo UI.
type ApplyAuditStore interface {
	Append(ctx context.Context, rec ApplyAuditRecord) error
	// ListRecent returns newest-first rows for projectID. When sceneID is non-empty, only rows with that sceneId are returned.
	ListRecent(ctx context.Context, projectID, sceneID string, limit int) ([]ApplyAuditListRow, error)
}
