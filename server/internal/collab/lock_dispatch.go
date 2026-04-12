package collab

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/reearth/reearth/server/pkg/id"
)

type lockMessage struct {
	Action   string `json:"action"`   // acquire | release | heartbeat
	Resource string `json:"resource"` // layer | widget
	ID       string `json:"id"`
}

func dispatchLock(ctx context.Context, hub *Hub, from *Conn, d json.RawMessage) error {
	var lm lockMessage
	if err := json.Unmarshal(d, &lm); err != nil {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_lock", "message": err.Error()}})
		return nil
	}
	if from.userID == "" {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "user required for locks"}})
		return nil
	}
	if from.operator == nil || !from.operator.IsWritableScene(from.sceneID) {
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "forbidden", "message": "scene not writable"}})
		return nil
	}

	switch lm.Resource {
	case "layer":
		if _, err := id.NLSLayerIDFrom(lm.ID); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_id", "message": err.Error()}})
			return nil
		}
	case "widget":
		if _, err := id.WidgetIDFrom(lm.ID); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_id", "message": err.Error()}})
			return nil
		}
	case "scene":
		if _, err := id.SceneIDFrom(lm.ID); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_id", "message": err.Error()}})
			return nil
		}
	case "widget_area":
		if !validWidgetAreaLockID(lm.ID) {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_id", "message": "invalid widget_area id (expected zone:section:area)"}})
			return nil
		}
	case "style":
		if _, err := id.StyleIDFrom(lm.ID); err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_id", "message": err.Error()}})
			return nil
		}
	default:
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_resource", "message": lm.Resource}})
		return nil
	}

	ttl := hub.lockTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	switch lm.Action {
	case "acquire":
		ok, holderWire, until, err := hub.tryLockAcquire(ctx, from.projectID, lm.Resource, lm.ID, from.userID, from.clientID, ttl)
		if err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_redis", "message": err.Error()}})
			return nil
		}
		if !ok {
			hu, hc := ParseLockHolderWire(holderWire)
			denied := map[string]any{"resource": lm.Resource, "id": lm.ID, "holderUserId": hu, "until": until.Format(time.RFC3339Nano)}
			if hc != "" {
				denied["holderClientId"] = hc
			}
			from.enqueueJSON(serverMessage{V: 1, T: "lock_denied", D: denied})
			return nil
		}
		hub.broadcastLockChanged(ctx, from.projectID, lm.Resource, lm.ID, holderWire, until)
		return nil
	case "release":
		ok, err := hub.tryLockRelease(ctx, from.projectID, lm.Resource, lm.ID, from.userID, from.clientID)
		if err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_redis", "message": err.Error()}})
			return nil
		}
		if !ok {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_release_failed", "message": "not holder or lock missing"}})
			return nil
		}
		hub.broadcastLockReleased(ctx, from.projectID, lm.Resource, lm.ID)
		return nil
	case "heartbeat":
		ok, until, err := hub.tryLockHeartbeat(ctx, from.projectID, lm.Resource, lm.ID, from.userID, from.clientID, ttl)
		if err != nil {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_redis", "message": err.Error()}})
			return nil
		}
		if !ok {
			from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "lock_heartbeat_failed", "message": "not holder or lock expired"}})
			return nil
		}
		hub.broadcastLockChanged(ctx, from.projectID, lm.Resource, lm.ID, LockHolderWire(from.userID, from.clientID), until)
		return nil
	default:
		from.enqueueJSON(serverMessage{V: 1, T: "error", D: map[string]string{"code": "invalid_action", "message": lm.Action}})
		return nil
	}
}

func (h *Hub) broadcastLockChanged(ctx context.Context, projectID, resource, rid, holderWire string, until time.Time) {
	hu, hc := ParseLockHolderWire(holderWire)
	d := map[string]any{
		"resource": resource, "id": rid, "holderUserId": hu,
		"until": until.Format(time.RFC3339Nano),
	}
	if hc != "" {
		d["holderClientId"] = hc
	}
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "lock_changed",
		D: d,
	})
	if err != nil {
		return
	}
	h.fanoutRoom(ctx, projectID, b)
}

func validWidgetAreaLockID(id string) bool {
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		return false
	}
	zones := map[string]bool{"inner": true, "outer": true}
	sections := map[string]bool{"left": true, "center": true, "right": true}
	areas := map[string]bool{"top": true, "middle": true, "bottom": true}
	return zones[parts[0]] && sections[parts[1]] && areas[parts[2]]
}

func (h *Hub) broadcastLockReleased(ctx context.Context, projectID, resource, rid string) {
	b, err := json.Marshal(serverMessage{
		V: 1,
		T: "lock_changed",
		D: map[string]any{"resource": resource, "id": rid, "released": true},
	})
	if err != nil {
		return
	}
	h.fanoutRoom(ctx, projectID, b)
}
