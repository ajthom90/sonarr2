package library

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
)

// recordingBus is a test Bus that records every published event for later
// assertions. It wraps a real events.Bus so handlers still run normally.
type recordingBus struct {
	inner events.Bus
	mu    sync.Mutex
	seen  []any
}

func newRecordingBus() *recordingBus {
	return &recordingBus{inner: events.NewBus(4)}
}

func (r *recordingBus) Publish(ctx context.Context, event any) error {
	r.mu.Lock()
	r.seen = append(r.seen, event)
	r.mu.Unlock()
	return r.inner.Publish(ctx, event)
}

// events returns a copy of all events seen so far, optionally filtered by
// concrete type.
func (r *recordingBus) events() []any {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]any, len(r.seen))
	copy(out, r.seen)
	return out
}

// ofType returns all events matching type T.
func eventsOfType[T any](r *recordingBus) []T {
	all := r.events()
	var out []T
	for _, e := range all {
		if t, ok := e.(T); ok {
			out = append(out, t)
		}
	}
	return out
}

// recordingBus satisfies events.Bus because register is unexported on
// events.Bus — but we can't delegate to inner.register from outside the
// package. Instead, recordingBus does NOT satisfy events.Bus directly;
// store tests that need a recorder use the wrapper pattern below.
//
// Concrete approach: use recordingBus.inner (a real Bus) as the store's
// bus. Subscribe a test handler that appends to r.seen. That way we
// observe events without trying to satisfy Bus.register.
//
// Helper: newRecorder returns (bus, getEvents) where bus is a real
// events.Bus and getEvents returns captured events filtered by type.
func newRecorder() (events.Bus, func() []any) {
	bus := events.NewBus(4)
	var mu sync.Mutex
	var seen []any
	// Subscribe a universal recorder for each known library event type.
	// New event types must be added here as they're introduced.
	recorder := func(e any) {
		mu.Lock()
		seen = append(seen, e)
		mu.Unlock()
	}
	events.SubscribeSync[SeriesAdded](bus, func(_ context.Context, e SeriesAdded) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[SeriesUpdated](bus, func(_ context.Context, e SeriesUpdated) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[SeriesDeleted](bus, func(_ context.Context, e SeriesDeleted) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[SeasonUpdated](bus, func(_ context.Context, e SeasonUpdated) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[EpisodeAdded](bus, func(_ context.Context, e EpisodeAdded) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[EpisodeUpdated](bus, func(_ context.Context, e EpisodeUpdated) error {
		recorder(e)
		return nil
	})
	events.SubscribeSync[EpisodeDeleted](bus, func(_ context.Context, e EpisodeDeleted) error {
		recorder(e)
		return nil
	})
	return bus, func() []any {
		mu.Lock()
		defer mu.Unlock()
		out := make([]any, len(seen))
		copy(out, seen)
		return out
	}
}

// filterEvents returns only events of type T from the captured slice.
func filterEvents[T any](events []any) []T {
	var out []T
	for _, e := range events {
		if typed, ok := e.(T); ok {
			out = append(out, typed)
		}
	}
	return out
}

// setupSQLiteLibrary returns a Library backed by an in-memory SQLite pool
// with all migrations applied. The bus is set via a recorder so tests can
// inspect published events via getEvents.
func setupSQLiteLibrary(t *testing.T) (*Library, *db.SQLitePool, events.Bus, func() []any) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	bus, getEvents := newRecorder()
	lib, err := New(pool, bus)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return lib, pool, bus, getEvents
}

// assertHasEvent asserts that at least one event of type T was captured.
func assertHasEvent[T any](t *testing.T, getEvents func() []any) T {
	t.Helper()
	matches := filterEvents[T](getEvents())
	if len(matches) == 0 {
		var zero T
		t.Errorf("expected at least one %T event; got none", zero)
		return zero
	}
	return matches[0]
}

// unused import guard: reflect is imported above to silence golangci-lint
// warnings if it becomes unused after later edits; keep it referenced.
var _ = reflect.TypeOf
