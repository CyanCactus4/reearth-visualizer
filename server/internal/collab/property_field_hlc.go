package collab

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const propertyFieldHlcRedisPrefix = collabChannelPrefix + "pfhlc:"

// propertyFieldHLCMemory stores last committed HLC per composite property field key.
type propertyFieldHLCMemory struct {
	mu sync.Mutex
	m  map[string]HLC
}

func newPropertyFieldHLCMemory() *propertyFieldHLCMemory {
	return &propertyFieldHLCMemory{m: make(map[string]HLC)}
}

func (s *propertyFieldHLCMemory) get(k string) HLC {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.m[k]
}

func (s *propertyFieldHLCMemory) set(k string, h HLC) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[k] = h
}

// PropertyFieldHLC returns the last committed HLC for a field (ZeroHLC if none).
func (h *Hub) PropertyFieldHLC(sceneID, propertyID, schemaGroupID, itemID, fieldID string) HLC {
	if h == nil || sceneID == "" || propertyID == "" || fieldID == "" {
		return ZeroHLC
	}
	k := propertyFieldCompositeKey(sceneID, propertyID, schemaGroupID, itemID, fieldID)
	if h.lockRedis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		raw, err := h.lockRedis.Get(ctx, propertyFieldHlcRedisPrefix+k).Bytes()
		if err == redis.Nil || len(raw) == 0 {
			return ZeroHLC
		}
		if err != nil {
			return ZeroHLC
		}
		v, err := hlcFromJSON(raw)
		if err != nil {
			return ZeroHLC
		}
		return v
	}
	if h.propertyFieldHLCMemory == nil {
		return ZeroHLC
	}
	return h.propertyFieldHLCMemory.get(k)
}

// writePropertyFieldHLC persists the authoritative HLC for a field (LWW register timestamp).
func (h *Hub) writePropertyFieldHLC(sceneID, propertyID, schemaGroupID, itemID, fieldID string, v HLC) {
	if h == nil || sceneID == "" || propertyID == "" || fieldID == "" {
		return
	}
	k := propertyFieldCompositeKey(sceneID, propertyID, schemaGroupID, itemID, fieldID)
	if h.lockRedis != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		b, err := hlcToJSON(v)
		if err != nil {
			return
		}
		_ = h.lockRedis.Set(ctx, propertyFieldHlcRedisPrefix+k, b, 0).Err()
		return
	}
	if h.propertyFieldHLCMemory == nil {
		h.propertyFieldHLCMemory = newPropertyFieldHLCMemory()
	}
	h.propertyFieldHLCMemory.set(k, v)
}

// advancePropertyFieldHLC merges incoming client HLC with stored, ticks forward, writes, returns broadcast stamp.
// Caller must hold hub.propertyCollabApplyMu (serializes with DB apply for this CRDT path).
func (h *Hub) advancePropertyFieldHLC(sceneID, propertyID, schemaGroupID, itemID, fieldID string, incoming HLC, nowMs int64) HLC {
	incoming.NodeID = normalizeNodeID(incoming.NodeID)
	cur := h.PropertyFieldHLC(sceneID, propertyID, schemaGroupID, itemID, fieldID)
	merged := MaxHLC(cur, incoming)
	next := merged.Tick(maxInt64(nowMs, merged.Physical))
	h.writePropertyFieldHLC(sceneID, propertyID, schemaGroupID, itemID, fieldID, next)
	return next
}

// normalizeNodeID trims and caps length (defensive).
func normalizeNodeID(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}
