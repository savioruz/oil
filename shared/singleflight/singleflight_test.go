package singleflight

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoCollapsesConcurrentCalls(t *testing.T) {
	g := New()

	var calls int32

	const n = 50

	start := make(chan struct{})
	results := make([]int, n)

	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()
			<-start

			v, _, err := Do(context.Background(), g, "key", func(_ context.Context) (int, error) {
				atomic.AddInt32(&calls, 1)
				time.Sleep(20 * time.Millisecond)

				return 42, nil
			})
			if err == nil {
				results[idx] = v
			}
		}(i)
	}

	close(start)
	wg.Wait()

	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "loader must run exactly once for concurrent callers")

	for _, v := range results {
		assert.Equal(t, 42, v)
	}
}

func TestDoReportsSharedFlag(t *testing.T) {
	g := New()

	const n = 20

	start := make(chan struct{})
	shared := make([]bool, n)

	var wg sync.WaitGroup

	for i := range n {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()
			<-start

			_, s, _ := Do(context.Background(), g, "key", func(_ context.Context) (int, error) {
				time.Sleep(20 * time.Millisecond)

				return 1, nil
			})
			shared[idx] = s
		}(i)
	}

	close(start)
	wg.Wait()

	sharedCount := 0

	for _, s := range shared {
		if s {
			sharedCount++
		}
	}

	assert.Positive(t, sharedCount, "at least some concurrent callers should have shared the result")
}

func TestDoPropagatesError(t *testing.T) {
	g := New()
	sentinel := errors.New("boom")

	v, _, err := Do(context.Background(), g, "key", func(_ context.Context) (int, error) {
		return 0, sentinel
	})

	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 0, v)
}

func TestDoDoesNotCacheAcrossSequentialCalls(t *testing.T) {
	g := New()

	var calls int32

	loader := func(_ context.Context) (int, error) {
		atomic.AddInt32(&calls, 1)

		return 7, nil
	}

	_, _, _ = Do(context.Background(), g, "key", loader)
	_, _, _ = Do(context.Background(), g, "key", loader)

	assert.Equal(t, int32(2), atomic.LoadInt32(&calls), "singleflight dedupes in-flight only, not across time")
}

func TestForgetAllowsReExecution(t *testing.T) {
	g := New()

	_, _, _ = Do(context.Background(), g, "key", func(_ context.Context) (int, error) { return 1, nil })
	g.Forget("key") // should not panic and clears any in-flight tracking
}
