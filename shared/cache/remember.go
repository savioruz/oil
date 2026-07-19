package cache

import (
	"context"
	"oil/shared/singleflight"

	"github.com/rs/zerolog/log"
)

// Remember implements the cache-aside pattern with stampede protection.
//
// It first checks the cache for key. On a miss, concurrent callers for the same
// key are collapsed by singleflight so loader (and therefore the underlying
// datastore) runs at most once; the resulting value is written back to the
// cache before being returned to every caller. Loader errors are propagated and
// never cached.
//
// The cache write uses a cancellation-detached context so an aborted request
// does not leave the cache unpopulated for the callers that shared the load.
func Remember[T any](
	ctx context.Context,
	c RedisCache,
	sf *singleflight.Group,
	key string,
	ttl int,
	loader func(context.Context) (T, error),
) (T, error) {
	var cached T
	if err := c.Get(ctx, key, &cached); err == nil {
		return cached, nil
	}

	val, _, err := singleflight.Do(ctx, sf, key, func(ctx context.Context) (T, error) {
		loaded, loadErr := loader(ctx)
		if loadErr != nil {
			return loaded, loadErr
		}

		saveCtx := context.WithoutCancel(ctx)
		if saveErr := c.Save(saveCtx, key, loaded, ttl); saveErr != nil {
			log.Error().Err(saveErr).Str("key", key).Msg("failed to save value to cache")
		}

		return loaded, nil
	})

	return val, err
}
