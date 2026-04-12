package collab

import (
	"context"
	"time"
)

// SubscribeSceneRevision receives scene updatedAt (Unix ms) after each successful collab apply on that scene.
// Caller must call cancel when done to avoid leaks.
func (h *Hub) SubscribeSceneRevision(sceneID string) (ch <-chan int64, cancel func()) {
	if h == nil || sceneID == "" {
		ch0 := make(chan int64)
		close(ch0)
		return ch0, func() {}
	}
	c := make(chan int64, 32)
	h.sceneRevSubMu.Lock()
	h.sceneRevSubs[sceneID] = append(h.sceneRevSubs[sceneID], c)
	h.sceneRevSubMu.Unlock()
	return c, func() {
		h.sceneRevSubMu.Lock()
		defer h.sceneRevSubMu.Unlock()
		arr := h.sceneRevSubs[sceneID]
		out := arr[:0]
		for _, x := range arr {
			if x != c {
				out = append(out, x)
			}
		}
		if len(out) == 0 {
			delete(h.sceneRevSubs, sceneID)
		} else {
			h.sceneRevSubs[sceneID] = out
		}
	}
}

func (h *Hub) deliverSceneRevisionSubscribers(sceneID string, rev int64) {
	if h == nil || sceneID == "" || rev == 0 {
		return
	}
	h.sceneRevSubMu.Lock()
	subs := append([]chan int64(nil), h.sceneRevSubs[sceneID]...)
	h.sceneRevSubMu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- rev:
		default:
		}
	}
}

// PublishSceneRevision notifies GraphQL/SSE subscribers after scene data changed (collab apply,
// GraphQL mutation, admin tools). No-op when hub, scene id, or rev is invalid.
func (h *Hub) PublishSceneRevision(sceneID string, rev int64) {
	h.publishSceneRevision(sceneID, rev)
}

// publishSceneRevision notifies local GraphQL/SSE subscribers and, when Redis relay is configured,
// other API instances (which deliver to their own local subscribers only).
func (h *Hub) publishSceneRevision(sceneID string, rev int64) {
	if h == nil || sceneID == "" || rev == 0 {
		return
	}
	h.deliverSceneRevisionSubscribers(sceneID, rev)
	if h.relay != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = h.relay.publishSceneRev(ctx, sceneID, rev, h.instanceID)
	}
}

// NotifyUserInRoom delivers a JSON message to every tab of targetUserId in projectID (best-effort).
func (h *Hub) NotifyUserInRoom(projectID, targetUserID string, payload []byte) {
	if h == nil || projectID == "" || targetUserID == "" || len(payload) == 0 {
		return
	}
	h.mu.RLock()
	r, ok := h.rooms[projectID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for c := range r.conns {
		if c.userID != targetUserID {
			continue
		}
		select {
		case c.send <- payload:
		default:
		}
	}
}
