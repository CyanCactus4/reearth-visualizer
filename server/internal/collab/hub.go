package collab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/reearth/reearthx/log"
	"golang.org/x/time/rate"
)

// Hub routes messages between WebSocket clients in the same project room.
// When Redis is configured, messages are also relayed to other server instances.
type Hub struct {
	mu sync.RWMutex

	instanceID string
	rooms      map[string]*room // key: project ID string

	relay *redisRelay

	locks     *lockTable
	lockTTL   time.Duration
	lockRedis *redis.Client // same as relay client when Redis is enabled; distributed locks

	chatMaxRunes int
	chatEvery    time.Duration
	chatLimiters sync.Map // key: projectID + "\x00" + userID -> *rate.Limiter (created lazily)

	cursorEvery    time.Duration
	cursorLimiters sync.Map // projectID + "\x00" + userID

	activityTypingEvery time.Duration
	activityMoveEvery   time.Duration
	activityLimiters    sync.Map // projectID + "\x00" + userID + "\x00" + kind

	chatStore  ChatHistoryStore
	applyAudit ApplyAuditStore

	sceneRevSubMu sync.Mutex
	sceneRevSubs  map[string][]chan int64 // scene ID → subscribers (buffered chans)

	// Per-widget field LWW clocks (in-memory; resets on process restart).
	widgetClockMu sync.Mutex
	widgetClocks  map[string]int64

	// Per-property-field LWW clocks (same Redis client as widget clocks when configured).
	propertyFieldClockMu sync.Mutex
	propertyFieldClocks  map[string]int64

	// Per-property-field HLC (CRDT LWW register timestamps); in-memory when Redis absent.
	propertyFieldHLCMemory *propertyFieldHLCMemory

	// Per-property document clock for merge_property_json (CAS).
	propertyDocClockMu sync.Mutex
	propertyDocClocks  map[string]int64

	// Serializes property-field collab paths (integer LWW + HLC CRDT) vs Mongo apply on this instance.
	propertyCollabApplyMu sync.Mutex

	opStack            CollabOpStack
	sceneSnapshotStore SceneSnapshotStore
	snapMu             sync.Mutex
	snapLastAt         map[string]time.Time // scene ID → last snapshot attempt
	mentionWebhook     string
}

type room struct {
	mu    sync.Mutex
	conns map[*Conn]struct{}
}

func newRoom() *room {
	return &room{conns: make(map[*Conn]struct{})}
}

func NewHub(o Options) *Hub {
	ttl := o.lockTTL()
	h := &Hub{
		instanceID:   uuid.NewString(),
		rooms:        make(map[string]*room),
		locks:        newLockTable(),
		lockTTL:      ttl,
		chatMaxRunes: o.chatMaxRunes(),
		chatEvery:    o.chatMinInterval(),

		cursorEvery:            o.cursorMinInterval(),
		activityTypingEvery:    o.activityTypingInterval(),
		activityMoveEvery:      o.activityMoveInterval(),
		chatStore:              o.ChatHistory,
		applyAudit:             o.ApplyAudit,
		sceneRevSubs:           make(map[string][]chan int64),
		widgetClocks:           make(map[string]int64),
		propertyFieldClocks:    make(map[string]int64),
		propertyFieldHLCMemory: newPropertyFieldHLCMemory(),
		propertyDocClocks:      make(map[string]int64),
		opStack:                o.OpStack,
		sceneSnapshotStore:     o.SceneSnapshot,
		snapLastAt:             make(map[string]time.Time),
		mentionWebhook:         strings.TrimSpace(o.MentionWebhookURL),
	}
	if o.RedisURL != "" {
		if r := newRedisRelay(o.RedisURL, h.instanceID); r != nil {
			h.relay = r
			h.lockRedis = r.Client()
		}
	}
	return h
}

// chatAllow enforces per-user-per-project chat spacing (burst 1).
func (h *Hub) chatAllow(projectID, userID string) bool {
	if userID == "" {
		return false
	}
	key := projectID + "\x00" + userID
	lim, _ := h.chatLimiters.LoadOrStore(key, rate.NewLimiter(rate.Every(h.chatEvery), 1))
	return lim.(*rate.Limiter).Allow()
}

// cursorAllow limits cursor broadcasts per user per project (burst 1).
func (h *Hub) cursorAllow(projectID, userID string) bool {
	if userID == "" {
		return false
	}
	key := projectID + "\x00" + userID
	lim, _ := h.cursorLimiters.LoadOrStore(key, rate.NewLimiter(rate.Every(h.cursorEvery), 1))
	return lim.(*rate.Limiter).Allow()
}

// activityAllow limits typing/move hints per user per project and kind (burst 1).
func (h *Hub) activityAllow(projectID, userID, kind string) bool {
	if userID == "" {
		return false
	}
	every := h.activityMoveEvery
	if kind == "typing" {
		every = h.activityTypingEvery
	}
	key := projectID + "\x00" + userID + "\x00" + kind
	lim, _ := h.activityLimiters.LoadOrStore(key, rate.NewLimiter(rate.Every(every), 1))
	return lim.(*rate.Limiter).Allow()
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
	h.presenceBroadcast(context.Background(), c, "join")
}

func (h *Hub) unregister(c *Conn) {
	h.presenceBroadcast(context.Background(), c, "leave")

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

func (h *Hub) relayPublish(ctx context.Context, projectID string, inner []byte) {
	if h.relay == nil {
		return
	}
	w := relayWire{I: h.instanceID, P: projectID, D: base64.StdEncoding.EncodeToString(inner)}
	b, err := json.Marshal(w)
	if err != nil {
		return
	}
	if err := h.relay.publish(ctx, projectID, b); err != nil {
		log.Warnfc(ctx, "collab: redis publish: %v", err)
	}
}

func (h *Hub) broadcastFromClient(ctx context.Context, projectID string, payload []byte, from *Conn) {
	h.broadcastLocal(projectID, payload, from)
	h.relayPublish(ctx, projectID, payload)
}

func (h *Hub) fanoutRoom(ctx context.Context, projectID string, inner []byte) {
	h.broadcastLocal(projectID, inner, nil)
	h.relayPublish(ctx, projectID, inner)
}

func (h *Hub) presenceBroadcast(ctx context.Context, c *Conn, event string) {
	uid := c.userID
	if uid == "" {
		uid = "unknown"
	}
	d := map[string]string{"event": event, "userId": uid}
	if c.photoURL != "" {
		d["photoURL"] = c.photoURL
	}
	b, err := json.Marshal(map[string]any{
		"v": 1,
		"t": "presence",
		"d": d,
	})
	if err != nil {
		return
	}
	h.fanoutRoom(ctx, c.projectID, b)
}

func (h *Hub) deliverFromRedis(_ context.Context, projectID string, payload []byte) {
	h.broadcastLocal(projectID, payload, nil)
}
