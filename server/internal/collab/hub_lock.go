package collab

import (
	"context"
	"time"
)

// LockHolder returns the active lock holder for a resource in the project room, if any.
func (h *Hub) LockHolder(ctx context.Context, projectID, resource, resourceID string) (holder string, active bool, err error) {
	if h.lockRedis != nil {
		return redisLockGet(ctx, h.lockRedis, projectID, resource, resourceID)
	}
	hh, ok := h.locks.Lookup(projectID, resource, resourceID)
	return hh, ok, nil
}

func (h *Hub) tryLockAcquire(ctx context.Context, projectID, resource, resourceID, userID, clientID string, ttl time.Duration) (ok bool, holder string, until time.Time, err error) {
	if h.lockRedis != nil {
		return redisLockTryAcquire(ctx, h.lockRedis, projectID, resource, resourceID, userID, clientID, ttl)
	}
	ok, holder, until = h.locks.TryAcquire(projectID, resource, resourceID, userID, clientID, ttl)
	return ok, holder, until, nil
}

func (h *Hub) tryLockRelease(ctx context.Context, projectID, resource, resourceID, userID, clientID string) (ok bool, err error) {
	if h.lockRedis != nil {
		return redisLockRelease(ctx, h.lockRedis, projectID, resource, resourceID, userID, clientID)
	}
	return h.locks.Release(projectID, resource, resourceID, userID, clientID), nil
}

func (h *Hub) tryLockHeartbeat(ctx context.Context, projectID, resource, resourceID, userID, clientID string, ttl time.Duration) (ok bool, until time.Time, err error) {
	if h.lockRedis != nil {
		return redisLockHeartbeat(ctx, h.lockRedis, projectID, resource, resourceID, userID, clientID, ttl)
	}
	ok, until = h.locks.Heartbeat(projectID, resource, resourceID, userID, clientID, ttl)
	return ok, until, nil
}
