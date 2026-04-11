package collab

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/reearth/reearthx/log"
)

const collabChannelPrefix = "collab:"

// sceneRevWire is published on channel collab:srev:<sceneID> for cross-instance scene revision fan-out.
type sceneRevWire struct {
	I string `json:"i"` // publisher instance id (skip echo to same instance)
	R int64  `json:"r"` // scene updatedAt unix ms
}

type redisRelay struct {
	client     *redis.Client
	instanceID string
}

func (r *redisRelay) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

func newRedisRelay(url, instanceID string) *redisRelay {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil
	}
	return &redisRelay{client: redis.NewClient(opts), instanceID: instanceID}
}

func (r *redisRelay) channel(projectID string) string {
	return collabChannelPrefix + projectID
}

func (r *redisRelay) publish(ctx context.Context, projectID string, body []byte) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Publish(ctx, r.channel(projectID), body).Err()
}

func (r *redisRelay) publishSceneRev(ctx context.Context, sceneID string, rev int64, instanceID string) error {
	if r == nil || r.client == nil || sceneID == "" || rev == 0 {
		return nil
	}
	b, err := json.Marshal(sceneRevWire{I: instanceID, R: rev})
	if err != nil {
		return nil
	}
	ch := collabChannelPrefix + "srev:" + sceneID
	return r.client.Publish(ctx, ch, b).Err()
}

func (r *redisRelay) startSubscriber(ctx context.Context, h *Hub) error {
	if r == nil || r.client == nil {
		return nil
	}
	if err := r.client.Ping(ctx).Err(); err != nil {
		return err
	}

	pubsub := r.client.PSubscribe(ctx, collabChannelPrefix+"*")
	go func() {
		defer func() { _ = pubsub.Close() }()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if msg == nil {
					continue
				}
				if strings.HasPrefix(msg.Channel, collabChannelPrefix+"srev:") {
					sceneID := strings.TrimPrefix(msg.Channel, collabChannelPrefix+"srev:")
					if sceneID == "" {
						continue
					}
					var sv sceneRevWire
					if err := json.Unmarshal([]byte(msg.Payload), &sv); err != nil {
						continue
					}
					if sv.I == r.instanceID {
						continue
					}
					h.deliverSceneRevisionSubscribers(sceneID, sv.R)
					continue
				}
				var w relayWire
				if err := json.Unmarshal([]byte(msg.Payload), &w); err != nil {
					continue
				}
				if w.I == r.instanceID {
					continue
				}
				raw, err := base64.StdEncoding.DecodeString(w.D)
				if err != nil {
					continue
				}
				pid := strings.TrimPrefix(msg.Channel, collabChannelPrefix)
				if pid == "" {
					pid = w.P
				}
				h.deliverFromRedis(ctx, pid, raw)
			}
		}
	}()

	// keepalive / reconnect is handled by redis client; log startup
	log.Infofc(ctx, "collab: redis pub/sub active instance=%s", r.instanceID)
	return nil
}
