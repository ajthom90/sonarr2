// Package events provides a typed generic event bus for in-process pub/sub.
//
// Because Go does not allow generic methods on interfaces, subscription is
// exposed as package-level generic functions (SubscribeSync, SubscribeAsync,
// SubscribeOrdered) that register handlers via an unexported method on the
// Bus interface.
package events

import (
	"context"
	"reflect"
	"sort"
	"sync"
)

// Bus dispatches events to registered handlers. Concrete buses are returned
// by NewBus and NewNoopBus.
type Bus interface {
	// Publish runs all sync handlers for the event's type in order, then
	// schedules async handlers on a bounded goroutine pool. Returns the
	// first sync handler error, if any. Sync handlers run even after one
	// errors — the caller sees only the first error but all handlers get
	// their chance to run.
	Publish(ctx context.Context, event any) error

	// register adds a handler for the given event type. Unexported so
	// external packages cannot implement Bus — use NewBus or NewNoopBus.
	register(eventType reflect.Type, h handlerEntry)
}

// handlerEntry is an internal registration record.
type handlerEntry struct {
	order int
	async bool
	fn    func(ctx context.Context, event any) error
}

// bus is the default in-process Bus implementation.
type bus struct {
	mu        sync.RWMutex
	handlers  map[reflect.Type][]handlerEntry
	asyncPool chan struct{}
}

// NewBus returns a new event bus. maxAsync bounds the number of concurrent
// async handler goroutines; values <= 0 default to 16.
func NewBus(maxAsync int) Bus {
	if maxAsync <= 0 {
		maxAsync = 16
	}
	return &bus{
		handlers:  make(map[reflect.Type][]handlerEntry),
		asyncPool: make(chan struct{}, maxAsync),
	}
}

// Publish implements Bus.
func (b *bus) Publish(ctx context.Context, event any) error {
	t := reflect.TypeOf(event)

	b.mu.RLock()
	entries := b.handlers[t]
	// Copy to avoid holding the read lock while running handlers.
	snapshot := make([]handlerEntry, len(entries))
	copy(snapshot, entries)
	b.mu.RUnlock()

	// Sync handlers run in order. We run every sync handler even if an
	// earlier one errors — callers may rely on side effects of later
	// handlers (e.g., history recording) regardless of whether the
	// first errored. The returned error is the first sync error.
	var firstErr error
	for _, h := range snapshot {
		if h.async {
			continue
		}
		if err := h.fn(ctx, event); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}

	// Async handlers dispatched to bounded pool. If the pool is full,
	// run inline rather than spawning unbounded goroutines or dropping.
	for _, h := range snapshot {
		if !h.async {
			continue
		}
		select {
		case b.asyncPool <- struct{}{}:
			go func(h handlerEntry) {
				defer func() { <-b.asyncPool }()
				_ = h.fn(ctx, event)
			}(h)
		default:
			_ = h.fn(ctx, event)
		}
	}

	return nil
}

// register implements Bus.register. It appends the handler to the slice
// for this event type and re-sorts by order so sync handlers run in the
// declared sequence.
func (b *bus) register(eventType reflect.Type, h handlerEntry) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entries := append(b.handlers[eventType], h)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].order < entries[j].order
	})
	b.handlers[eventType] = entries
}
