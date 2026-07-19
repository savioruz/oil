package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	otelMocks "oil/infras/otel/mocks"
	"oil/shared/cache"
	"oil/shared/singleflight"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rememberPayload struct {
	Name string `json:"name"`
}

func newRememberDeps(t *testing.T) (cache.RedisCache, *singleflight.Group, *miniredis.Miniredis) {
	t.Helper()

	srv, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(srv.Close)

	client := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	return cache.New(client, otelMocks.NewOtel()), singleflight.New(), srv
}

func TestRememberReturnsCachedValueWithoutLoading(t *testing.T) {
	ctx := context.Background()
	c, sf, _ := newRememberDeps(t)

	require.NoError(t, c.Save(ctx, "k", rememberPayload{Name: "cached"}, 60))

	var loaded int32

	got, err := cache.Remember(ctx, c, sf, "k", 60, func(_ context.Context) (rememberPayload, error) {
		atomic.AddInt32(&loaded, 1)

		return rememberPayload{Name: "loaded"}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "cached", got.Name)
	assert.Equal(t, int32(0), atomic.LoadInt32(&loaded), "loader must not run on cache hit")
}

func TestRememberLoadsAndStoresOnMiss(t *testing.T) {
	ctx := context.Background()
	c, sf, srv := newRememberDeps(t)

	got, err := cache.Remember(ctx, c, sf, "k", 60, func(_ context.Context) (rememberPayload, error) {
		return rememberPayload{Name: "loaded"}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, "loaded", got.Name)

	// Value must be written to cache for subsequent reads.
	assert.True(t, srv.Exists("k"), "loaded value must be persisted to cache")

	var fromCache rememberPayload
	require.NoError(t, c.Get(ctx, "k", &fromCache))
	assert.Equal(t, "loaded", fromCache.Name)
}

func TestRememberCollapsesStampede(t *testing.T) {
	ctx := context.Background()
	c, sf, _ := newRememberDeps(t)

	var loads int32

	const n = 50

	start := make(chan struct{})

	var wg sync.WaitGroup

	for range n {
		wg.Add(1)

		go func() {
			defer wg.Done()
			<-start

			_, _ = cache.Remember(ctx, c, sf, "k", 60, func(_ context.Context) (rememberPayload, error) {
				atomic.AddInt32(&loads, 1)
				time.Sleep(20 * time.Millisecond)

				return rememberPayload{Name: "loaded"}, nil
			})
		}()
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&loads), "loader must run once for a concurrent stampede")
}

func TestRememberDoesNotCacheErrors(t *testing.T) {
	ctx := context.Background()
	c, sf, srv := newRememberDeps(t)

	sentinel := errors.New("db down")

	_, err := cache.Remember(ctx, c, sf, "k", 60, func(_ context.Context) (rememberPayload, error) {
		return rememberPayload{}, sentinel
	})

	require.ErrorIs(t, err, sentinel)
	assert.False(t, srv.Exists("k"), "failed loads must not be cached")
}
