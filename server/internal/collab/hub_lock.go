package collab

import (
	"context"
	"time"
)

func (h *Hub) tryLockAcquire(ctx context.Context, projectID, resource, resourceID, userID string, ttl time.Duration) (ok bool, holder string, until time.Time, err error) {
	if h.lockRedis != nil {
		return redisLockTryAcquire(ctx, h.lockRedis, projectID, resource, resourceID, userID, ttl)
	}
	ok, holder, until = h.locks.TryAcquire(projectID, resource, resourceID, userID, ttl)
	return ok, holder, until, nil
}

func (h *Hub) tryLockRelease(ctx context.Context, projectID, resource, resourceID, userID string) (ok bool, err error) {
	if h.lockRedis != nil {
		return redisLockRelease(ctx, h.lockRedis, projectID, resource, resourceID, userID)
	}
	return h.locks.Release(projectID, resource, resourceID, userID), nil
}

func (h *Hub) tryLockHeartbeat(ctx context.Context, projectID, resource, resourceID, userID string, ttl time.Duration) (ok bool, until time.Time, err error) {
	if h.lockRedis != nil {
		return redisLockHeartbeat(ctx, h.lockRedis, projectID, resource, resourceID, userID, ttl)
	}
	ok, until = h.locks.Heartbeat(projectID, resource, resourceID, userID, ttl)
	return ok, until, nil
}
