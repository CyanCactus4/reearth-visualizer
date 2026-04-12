package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisLockReleaseScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("del", KEYS[1])
else
  return 0
end
`
	redisLockHeartbeatScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
  return redis.call("expire", KEYS[1], tonumber(ARGV[2]))
else
  return 0
end
`
)

func redisLockKey(projectID, resource, resourceID string) string {
	return "collab:lock:" + projectID + ":" + resource + ":" + resourceID
}

func redisLockTTLSeconds(ttl time.Duration) int {
	sec := int(ttl / time.Second)
	if sec < 1 {
		return 1
	}
	return sec
}

func redisLockGet(ctx context.Context, rdb *redis.Client, projectID, resource, resourceID string) (holder string, active bool, err error) {
	key := redisLockKey(projectID, resource, resourceID)
	s, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if s == "" {
		return "", false, nil
	}
	return s, true, nil
}

// redisLockTryAcquire uses SET NX EX; same holder may refresh with EXPIRE.
func redisLockTryAcquire(ctx context.Context, rdb *redis.Client, projectID, resource, resourceID, userID, clientID string, ttl time.Duration) (granted bool, holder string, until time.Time, err error) {
	wire := LockHolderWire(userID, clientID)
	key := redisLockKey(projectID, resource, resourceID)
	ok, err := rdb.SetNX(ctx, key, wire, ttl).Result()
	if err != nil {
		return false, "", time.Time{}, err
	}
	if ok {
		return true, wire, time.Now().Add(ttl), nil
	}
	cur, err := rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		// Lost race; retry once.
		ok2, err2 := rdb.SetNX(ctx, key, wire, ttl).Result()
		if err2 != nil {
			return false, "", time.Time{}, err2
		}
		if ok2 {
			return true, wire, time.Now().Add(ttl), nil
		}
		cur, err = rdb.Get(ctx, key).Result()
		if err != nil {
			return false, "", time.Time{}, err
		}
	} else if err != nil {
		return false, "", time.Time{}, err
	}
	if cur == wire {
		if err := rdb.Expire(ctx, key, ttl).Err(); err != nil {
			return false, "", time.Time{}, err
		}
		rem, err := rdb.TTL(ctx, key).Result()
		if err != nil {
			return true, wire, time.Now().Add(ttl), nil
		}
		return true, wire, time.Now().Add(rem), nil
	}
	rem, _ := rdb.TTL(ctx, key).Result()
	if rem < 0 {
		rem = 0
	}
	return false, cur, time.Now().Add(rem), nil
}

func redisLockRelease(ctx context.Context, rdb *redis.Client, projectID, resource, resourceID, userID, clientID string) (bool, error) {
	wire := LockHolderWire(userID, clientID)
	key := redisLockKey(projectID, resource, resourceID)
	n, err := rdb.Eval(ctx, redisLockReleaseScript, []string{key}, wire).Int()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}

func redisLockHeartbeat(ctx context.Context, rdb *redis.Client, projectID, resource, resourceID, userID, clientID string, ttl time.Duration) (ok bool, until time.Time, err error) {
	wire := LockHolderWire(userID, clientID)
	key := redisLockKey(projectID, resource, resourceID)
	sec := redisLockTTLSeconds(ttl)
	n, err := rdb.Eval(ctx, redisLockHeartbeatScript, []string{key}, wire, sec).Int()
	if err != nil {
		return false, time.Time{}, err
	}
	if n != 1 {
		return false, time.Time{}, nil
	}
	rem, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return true, time.Now().Add(ttl), nil
	}
	return true, time.Now().Add(rem), nil
}
