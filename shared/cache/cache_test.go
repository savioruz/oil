package cache

import (
	"context"
	"testing"

	otelMocks "github.com/savioruz/oil/infras/otel/mocks"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T) (RedisCache, *miniredis.Miniredis) {
	t.Helper()

	srv, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(srv.Close)

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return New(client, otelMocks.NewOtel()), srv
}

func TestClearRemovesOnlyMatchingKeys(t *testing.T) {
	ctx := context.Background()
	c, srv := newTestCache(t)

	require.NoError(t, srv.Set("todo:gets:1", "a"))
	require.NoError(t, srv.Set("todo:gets:2", "b"))
	require.NoError(t, srv.Set("other:1", "c"))

	require.NoError(t, c.Clear(ctx, "todo:gets:*"))

	assert.False(t, srv.Exists("todo:gets:1"))
	assert.False(t, srv.Exists("todo:gets:2"))
	assert.True(t, srv.Exists("other:1"), "non-matching keys must be preserved")
}

func TestClearNoMatchIsNoop(t *testing.T) {
	ctx := context.Background()
	c, srv := newTestCache(t)

	require.NoError(t, srv.Set("keep:1", "a"))

	require.NoError(t, c.Clear(ctx, "missing:*"))

	assert.True(t, srv.Exists("keep:1"))
}

func TestSaveThenGetRoundTrip(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t)

	type payload struct {
		Name string `json:"name"`
	}

	require.NoError(t, c.Save(ctx, "k:1", payload{Name: "alice"}, 60))

	var got payload
	require.NoError(t, c.Get(ctx, "k:1", &got))
	assert.Equal(t, "alice", got.Name)
}

func TestGetMissReturnsNil(t *testing.T) {
	ctx := context.Background()
	c, _ := newTestCache(t)

	var got string
	err := c.Get(ctx, "absent", &got)
	assert.ErrorIs(t, err, Nil)
}
