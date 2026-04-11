package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func widgetClockKey(sceneID, widgetID, field string) string {
	return sceneID + "\x00" + widgetID + "\x00" + field
}

func widgetClockRedisKey(k string) string {
	return collabChannelPrefix + "wfclk:" + k
}

// WidgetFieldClock returns the server LWW clock for a widget field (0 if unseen).
// When Redis is configured (same client as collab relay), clocks are shared across API replicas.
func (h *Hub) WidgetFieldClock(sceneID, widgetID, field string) int64 {
	if h == nil || sceneID == "" || widgetID == "" || field == "" {
		return 0
	}
	if h.lockRedis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		v, err := h.lockRedis.Get(ctx, widgetClockRedisKey(widgetClockKey(sceneID, widgetID, field))).Int64()
		if err == redis.Nil {
			return 0
		}
		if err != nil {
			return 0
		}
		return v
	}
	h.widgetClockMu.Lock()
	defer h.widgetClockMu.Unlock()
	return h.widgetClocks[widgetClockKey(sceneID, widgetID, field)]
}

// BumpWidgetFieldClocks increments clocks for the given fields and returns the new values.
func (h *Hub) BumpWidgetFieldClocks(sceneID, widgetID string, fields []string) map[string]int64 {
	out := make(map[string]int64)
	if h == nil || sceneID == "" || widgetID == "" {
		return out
	}
	if h.lockRedis != nil {
		return h.bumpWidgetFieldClocksRedis(sceneID, widgetID, fields)
	}
	h.widgetClockMu.Lock()
	defer h.widgetClockMu.Unlock()
	for _, f := range fields {
		if f == "" {
			continue
		}
		k := widgetClockKey(sceneID, widgetID, f)
		h.widgetClocks[k]++
		out[f] = h.widgetClocks[k]
	}
	return out
}

func (h *Hub) bumpWidgetFieldClocksRedis(sceneID, widgetID string, fields []string) map[string]int64 {
	out := make(map[string]int64)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	type pair struct {
		field string
		key   string
	}
	var pairs []pair
	for _, f := range fields {
		if f == "" {
			continue
		}
		pairs = append(pairs, pair{
			field: f,
			key:   widgetClockRedisKey(widgetClockKey(sceneID, widgetID, f)),
		})
	}
	if len(pairs) == 0 {
		return out
	}
	pipe := h.lockRedis.Pipeline()
	cmds := make([]*redis.IntCmd, len(pairs))
	for i := range pairs {
		cmds[i] = pipe.Incr(ctx, pairs[i].key)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return out
	}
	for i := range pairs {
		v, err := cmds[i].Result()
		if err != nil {
			continue
		}
		out[pairs[i].field] = v
	}
	return out
}
