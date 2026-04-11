package collab

import "time"

// Options configures the collaboration hub (WebSocket rooms, Redis relay, locks, chat).
type Options struct {
	RedisURL          string
	LockTTLSeconds    int
	ChatMaxRunes      int
	ChatMinIntervalMs int
	MaxMessageBytes   int
	MaxMessagesPerSec int

	// CursorMinIntervalMs: min spacing between cursor broadcasts per user (default 50).
	CursorMinIntervalMs int
	// ActivityTypingMinIntervalMs: min spacing for typing activity (default 2000).
	ActivityTypingMinIntervalMs int
	// ActivityMoveMinIntervalMs: min spacing for move activity (default 800).
	ActivityMoveMinIntervalMs int

	// ChatHistory persists chat lines (Mongo); nil skips persistence and GET /api/collab/chat.
	ChatHistory ChatHistoryStore
	// ApplyAudit appends successful apply ops (Mongo); nil skips journaling.
	ApplyAudit ApplyAuditStore
	// OpStack persists undo/redo for collab (Mongo); nil disables POST /api/collab/undo|redo.
	OpStack CollabOpStack
	// MentionWebhookURL optional HTTPS endpoint for out-of-room @mention delivery (POST JSON).
	MentionWebhookURL string
}

func (o Options) lockTTL() time.Duration {
	ttl := time.Duration(o.LockTTLSeconds) * time.Second
	if ttl <= 0 {
		return 5 * time.Minute
	}
	return ttl
}

func (o Options) chatMaxRunes() int {
	if o.ChatMaxRunes <= 0 {
		return 4000
	}
	return o.ChatMaxRunes
}

func (o Options) chatMinInterval() time.Duration {
	ms := o.ChatMinIntervalMs
	if ms <= 0 {
		ms = 1000
	}
	return time.Duration(ms) * time.Millisecond
}

func (o Options) cursorMinInterval() time.Duration {
	ms := o.CursorMinIntervalMs
	if ms <= 0 {
		ms = 50
	}
	return time.Duration(ms) * time.Millisecond
}

func (o Options) activityTypingInterval() time.Duration {
	ms := o.ActivityTypingMinIntervalMs
	if ms <= 0 {
		ms = 2000
	}
	return time.Duration(ms) * time.Millisecond
}

func (o Options) activityMoveInterval() time.Duration {
	ms := o.ActivityMoveMinIntervalMs
	if ms <= 0 {
		ms = 800
	}
	return time.Duration(ms) * time.Millisecond
}
