package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

func propertyFieldCompositeKey(sceneID, propertyID, schemaGroupID, itemID, fieldID string) string {
	return sceneID + "\x00" + propertyID + "\x00" + schemaGroupID + "\x00" + itemID + "\x00" + fieldID
}

func propertyFieldRedisKey(k string) string {
	return collabChannelPrefix + "pfclk:" + k
}

// PropertyFieldClock returns the server LWW clock for one property field pointer (0 if unseen).
func (h *Hub) PropertyFieldClock(sceneID, propertyID, schemaGroupID, itemID, fieldID string) int64 {
	if h == nil || sceneID == "" || propertyID == "" || fieldID == "" {
		return 0
	}
	k := propertyFieldCompositeKey(sceneID, propertyID, schemaGroupID, itemID, fieldID)
	if h.lockRedis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		v, err := h.lockRedis.Get(ctx, propertyFieldRedisKey(k)).Int64()
		if err == redis.Nil {
			return 0
		}
		if err != nil {
			return 0
		}
		return v
	}
	h.propertyFieldClockMu.Lock()
	defer h.propertyFieldClockMu.Unlock()
	return h.propertyFieldClocks[k]
}

// BumpPropertyFieldClock increments the clock for the given property field and returns the new value.
func (h *Hub) BumpPropertyFieldClock(sceneID, propertyID, schemaGroupID, itemID, fieldID string) int64 {
	if h == nil || sceneID == "" || propertyID == "" || fieldID == "" {
		return 0
	}
	k := propertyFieldCompositeKey(sceneID, propertyID, schemaGroupID, itemID, fieldID)
	if h.lockRedis != nil {
		return h.bumpPropertyFieldClockRedis(k)
	}
	h.propertyFieldClockMu.Lock()
	defer h.propertyFieldClockMu.Unlock()
	h.propertyFieldClocks[k]++
	return h.propertyFieldClocks[k]
}

func (h *Hub) bumpPropertyFieldClockRedis(k string) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	v, err := h.lockRedis.Incr(ctx, propertyFieldRedisKey(k)).Result()
	if err != nil {
		return 0
	}
	return v
}
