package collab

import (
	"sync"
	"time"
)

// lockTable holds per-project optimistic locks (single-process; use Redis for cross-node in future).
type lockTable struct {
	mu sync.Mutex
	m  map[string]*lockEntry // key: projectID/resource/id
}

type lockEntry struct {
	holder string
	until  time.Time
}

func newLockTable() *lockTable {
	return &lockTable{m: make(map[string]*lockEntry)}
}

func lockKey(projectID, resource, resourceID string) string {
	return projectID + "/" + resource + "/" + resourceID
}

func (t *lockTable) pruneLocked(now time.Time) {
	for k, e := range t.m {
		if e == nil || !e.until.After(now) {
			delete(t.m, k)
		}
	}
}

// TryAcquire returns granted=true if caller holds the lock after this call.
func (t *lockTable) TryAcquire(projectID, resource, resourceID, userID string, ttl time.Duration) (granted bool, holder string, until time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	t.pruneLocked(now)
	k := lockKey(projectID, resource, resourceID)
	if e, ok := t.m[k]; ok && e.until.After(now) {
		if e.holder == userID {
			e.until = now.Add(ttl)
			return true, userID, e.until
		}
		return false, e.holder, e.until
	}
	until = now.Add(ttl)
	t.m[k] = &lockEntry{holder: userID, until: until}
	return true, userID, until
}

func (t *lockTable) Release(projectID, resource, resourceID, userID string) (released bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	k := lockKey(projectID, resource, resourceID)
	e, ok := t.m[k]
	if !ok || !e.until.After(now) {
		delete(t.m, k)
		return true
	}
	if e.holder != userID {
		return false
	}
	delete(t.m, k)
	return true
}

// Heartbeat extends TTL for the current holder only.
func (t *lockTable) Heartbeat(projectID, resource, resourceID, userID string, ttl time.Duration) (ok bool, until time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now()
	k := lockKey(projectID, resource, resourceID)
	e, ok := t.m[k]
	if !ok || !e.until.After(now) {
		return false, time.Time{}
	}
	if e.holder != userID {
		return false, e.until
	}
	e.until = now.Add(ttl)
	return true, e.until
}
