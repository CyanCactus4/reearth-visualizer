package collab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/reearth/reearthx/log"
)

// Hub routes messages between WebSocket clients in the same project room.
// When Redis is configured, messages are also relayed to other server instances.
type Hub struct {
	mu sync.RWMutex

	instanceID string
	rooms      map[string]*room // key: project ID string

	relay *redisRelay
}

type room struct {
	mu    sync.Mutex
	conns map[*Conn]struct{}
}

func newRoom() *room {
	return &room{conns: make(map[*Conn]struct{})}
}

func NewHub(redisURL string) *Hub {
	h := &Hub{
		instanceID: uuid.NewString(),
		rooms:      make(map[string]*room),
	}
	if redisURL != "" {
		if r := newRedisRelay(redisURL, h.instanceID); r != nil {
			h.relay = r
		}
	}
	return h
}

// Run starts optional Redis subscriber. Call once at process startup.
func (h *Hub) Run(ctx context.Context) {
	if h.relay == nil {
		return
	}
	if err := h.relay.startSubscriber(ctx, h); err != nil {
		log.Errorfc(ctx, "collab: redis subscriber: %v", err)
	}
}

func (h *Hub) register(c *Conn) {
	h.mu.Lock()
	r, ok := h.rooms[c.projectID]
	if !ok {
		r = newRoom()
		h.rooms[c.projectID] = r
	}
	h.mu.Unlock()

	r.mu.Lock()
	r.conns[c] = struct{}{}
	n := len(r.conns)
	r.mu.Unlock()
	log.Infofc(context.Background(), "collab: join room project=%s conns=%d", c.projectID, n)
}

func (h *Hub) unregister(c *Conn) {
	h.mu.RLock()
	r, ok := h.rooms[c.projectID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.Lock()
	delete(r.conns, c)
	empty := len(r.conns) == 0
	r.mu.Unlock()

	if empty {
		h.mu.Lock()
		// re-fetch in case room was replaced (should not happen)
		if r2 := h.rooms[c.projectID]; r2 == r {
			delete(h.rooms, c.projectID)
		}
		h.mu.Unlock()
	}
	log.Infofc(context.Background(), "collab: leave room project=%s empty=%v", c.projectID, empty)
}

// broadcastLocal sends payload to every connection in the room except `except` (may be nil).
func (h *Hub) broadcastLocal(projectID string, payload []byte, except *Conn) {
	h.mu.RLock()
	r, ok := h.rooms[projectID]
	h.mu.RUnlock()
	if !ok {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for c := range r.conns {
		if except != nil && c == except {
			continue
		}
		select {
		case c.send <- payload:
		default:
			// slow consumer: drop outbound message
		}
	}
}

// relayWire is the JSON payload published to Redis.
type relayWire struct {
	I string `json:"i"` // instance id
	P string `json:"p"` // project id
	D string `json:"d"` // base64 payload
}

func (h *Hub) broadcastFromClient(ctx context.Context, projectID string, payload []byte, from *Conn) {
	h.broadcastLocal(projectID, payload, from)

	if h.relay == nil {
		return
	}
	w := relayWire{I: h.instanceID, P: projectID, D: base64.StdEncoding.EncodeToString(payload)}
	b, err := json.Marshal(w)
	if err != nil {
		return
	}
	if err := h.relay.publish(ctx, projectID, b); err != nil {
		log.Warnfc(ctx, "collab: redis publish: %v", err)
	}
}

func (h *Hub) deliverFromRedis(_ context.Context, projectID string, payload []byte) {
	h.broadcastLocal(projectID, payload, nil)
}
