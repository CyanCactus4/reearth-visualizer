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
