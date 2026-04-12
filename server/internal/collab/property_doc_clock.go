package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func propertyDocCompositeKey(sceneID, propertyID string) string {
	return sceneID + "\x00" + propertyID
}

func propertyDocRedisKey(k string) string {
	return collabChannelPrefix + "pdclk:" + k
}

// PropertyDocClock returns the server revision for merge_property_json CAS (0 if unseen).
func (h *Hub) PropertyDocClock(sceneID, propertyID string) int64 {
	if h == nil || sceneID == "" || propertyID == "" {
		return 0
	}
	k := propertyDocCompositeKey(sceneID, propertyID)
	if h.lockRedis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		v, err := h.lockRedis.Get(ctx, propertyDocRedisKey(k)).Int64()
		if err == redis.Nil {
			return 0
		}
		if err != nil {
			return 0
		}
		return v
	}
	h.propertyDocClockMu.Lock()
	defer h.propertyDocClockMu.Unlock()
	return h.propertyDocClocks[k]
}

// BumpPropertyDocClock increments the property JSON merge document clock.
func (h *Hub) BumpPropertyDocClock(sceneID, propertyID string) int64 {
	if h == nil || sceneID == "" || propertyID == "" {
		return 0
	}
	k := propertyDocCompositeKey(sceneID, propertyID)
	if h.lockRedis != nil {
		return h.bumpPropertyDocClockRedis(k)
	}
	h.propertyDocClockMu.Lock()
	defer h.propertyDocClockMu.Unlock()
	h.propertyDocClocks[k]++
	return h.propertyDocClocks[k]
}

func (h *Hub) bumpPropertyDocClockRedis(k string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	v, err := h.lockRedis.Incr(ctx, propertyDocRedisKey(k)).Result()
	if err != nil {
		return 0
	}
	return v
}
