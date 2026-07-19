// Package singleflight provides a generic, type-safe wrapper over
// golang.org/x/sync/singleflight for deduplicating concurrent work that shares
// a key (for example, collapsing a cache-stampede into a single load).
package singleflight

import (
	"context"

	"golang.org/x/sync/singleflight"
)

// Group deduplicates concurrent calls that share the same key. The zero value
// is not usable; construct one with New.
type Group struct {
	g singleflight.Group
}

// New creates a ready-to-use Group.
func New() *Group {
	return &Group{}
}

// Do executes fn once for a given key. Concurrent callers using the same key
// while a call is in flight share the single result instead of each running fn.
//
// shared reports whether the returned value was shared with other callers
// rather than produced by this call.
//
// Caveat: the shared execution runs with the first caller's context;
// golang.org/x/sync/singleflight does not merge contexts. This is acceptable
// for idempotent read paths, which is the intended use.
func Do[T any](ctx context.Context, g *Group, key string, fn func(context.Context) (T, error)) (val T, shared bool, err error) {
	result, err, shared := g.g.Do(key, func() (any, error) {
		return fn(ctx)
	})
	if err != nil {
		var zero T

		return zero, shared, err
	}

	typed, _ := result.(T)

	return typed, shared, nil
}

// Forget removes a key so the next Do re-executes fn instead of joining an
// in-flight call.
func (g *Group) Forget(key string) {
	g.g.Forget(key)
}
