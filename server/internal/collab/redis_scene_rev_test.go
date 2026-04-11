package collab

import (
	"context"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishSceneRevision_RedisFanout(t *testing.T) {
	s := miniredis.RunT(t)
	url := "redis://" + s.Addr()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h1 := NewHub(Options{RedisURL: url})
	h2 := NewHub(Options{RedisURL: url})
	require.NotNil(t, h1.relay)
	require.NotNil(t, h2.relay)

	h1.Run(ctx)
	h2.Run(ctx)
	time.Sleep(80 * time.Millisecond)

	ch, unsub := h2.SubscribeSceneRevision("scene-a")
	defer unsub()

	h1.publishSceneRevision("scene-a", 9001)

	select {
	case v := <-ch:
		assert.Equal(t, int64(9001), v)
	case <-time.After(3 * time.Second):
		t.Fatal("expected scene revision on peer hub via Redis")
	}
}

func TestWidgetFieldClocks_RedisShared(t *testing.T) {
	s := miniredis.RunT(t)
	url := "redis://" + s.Addr()
	h1 := NewHub(Options{RedisURL: url})
	h2 := NewHub(Options{RedisURL: url})
	require.NotNil(t, h1.lockRedis)

	out := h1.BumpWidgetFieldClocks("s", "w", []string{"enabled", "layout"})
	assert.Equal(t, int64(1), out["enabled"])
	assert.Equal(t, int64(1), out["layout"])

	assert.Equal(t, int64(1), h2.WidgetFieldClock("s", "w", "enabled"))
	out2 := h2.BumpWidgetFieldClocks("s", "w", []string{"enabled"})
	assert.Equal(t, int64(2), out2["enabled"])
	assert.Equal(t, int64(2), h1.WidgetFieldClock("s", "w", "enabled"))
}
