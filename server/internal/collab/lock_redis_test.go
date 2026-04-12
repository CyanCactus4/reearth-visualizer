package collab

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisLockAcquireContention(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer func() { _ = rdb.Close() }()

	ctx := context.Background()
	ttl := time.Minute

	ok, holder, _, err := redisLockTryAcquire(ctx, rdb, "p1", "layer", "L1", "alice", "", ttl)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "alice", holder)

	ok2, holder2, _, err := redisLockTryAcquire(ctx, rdb, "p1", "layer", "L1", "bob", "", ttl)
	require.NoError(t, err)
	assert.False(t, ok2)
	assert.Equal(t, "alice", holder2)

	released, err := redisLockRelease(ctx, rdb, "p1", "layer", "L1", "alice", "")
	require.NoError(t, err)
	assert.True(t, released)

	ok3, holder3, _, err := redisLockTryAcquire(ctx, rdb, "p1", "layer", "L1", "bob", "", ttl)
	require.NoError(t, err)
	assert.True(t, ok3)
	assert.Equal(t, "bob", holder3)
}

func TestRedisLockHeartbeat(t *testing.T) {
	s := miniredis.RunT(t)
	defer s.Close()
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer func() { _ = rdb.Close() }()
	ctx := context.Background()
	ttl := 30 * time.Second

	_, _, _, err := redisLockTryAcquire(ctx, rdb, "p", "widget", "w", "u1", "", ttl)
	require.NoError(t, err)
	ok, until, err := redisLockHeartbeat(ctx, rdb, "p", "widget", "w", "u1", "", ttl)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, until.After(time.Now()))
}
