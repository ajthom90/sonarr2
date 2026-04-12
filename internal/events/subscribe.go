package events

import (
	"context"
	"reflect"
)

// defaultSyncOrder is the order value used by SubscribeSync when the caller
// does not specify one. SubscribeOrdered lets callers override this.
const defaultSyncOrder = 100

// SubscribeSync registers a synchronous handler for events of type T.
// Sync handlers block Publish until they return; the first error is
// propagated back to the caller of Publish (later sync handlers still run).
func SubscribeSync[T any](bus Bus, handler func(ctx context.Context, e T) error) {
	var zero T
	t := reflect.TypeOf(zero)
	bus.register(t, handlerEntry{
		order: defaultSyncOrder,
		async: false,
		fn: func(ctx context.Context, event any) error {
			typed, ok := event.(T)
			if !ok {
				return nil
			}
			return handler(ctx, typed)
		},
	})
}

// SubscribeAsync registers an async handler for events of type T. Async
// handlers run on a bounded goroutine pool and their errors are discarded.
// Use async when the handler's side effect is fire-and-forget — notifications,
// UI broadcasts, cache warmups — and a failure should not block the publisher.
func SubscribeAsync[T any](bus Bus, handler func(ctx context.Context, e T)) {
	var zero T
	t := reflect.TypeOf(zero)
	bus.register(t, handlerEntry{
		order: 0,
		async: true,
		fn: func(ctx context.Context, event any) error {
			typed, ok := event.(T)
			if !ok {
				return nil
			}
			handler(ctx, typed)
			return nil
		},
	})
}

// SubscribeOrdered registers a synchronous handler that runs at a specific
// position in the sync handler chain. Lower order values run first. Use
// when two handlers have a required ordering relationship (e.g., stats
// recompute must run before notification dispatch for a single event).
func SubscribeOrdered[T any](bus Bus, order int, handler func(ctx context.Context, e T) error) {
	var zero T
	t := reflect.TypeOf(zero)
	bus.register(t, handlerEntry{
		order: order,
		async: false,
		fn: func(ctx context.Context, event any) error {
			typed, ok := event.(T)
			if !ok {
				return nil
			}
			return handler(ctx, typed)
		},
	})
}
