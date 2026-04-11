package collab

import "context"

// ChatMessageRecord is one persisted or live chat line (TASK.md collab chat).
type ChatMessageRecord struct {
	ID     string `json:"id"`
	UserID string `json:"userId"`
	Text   string `json:"text"`
	Ts     int64  `json:"ts"`
	// Mentions are @handles parsed server-side (without @), deduplicated, max 20.
	Mentions []string `json:"mentions,omitempty"`
}

// ChatHistoryStore persists and loads recent chat messages per project.
type ChatHistoryStore interface {
	Append(ctx context.Context, projectID, userID, text string, tsUnix int64, messageID string, mentions []string) error
	ListRecent(ctx context.Context, projectID string, limit int) ([]ChatMessageRecord, error)
}
