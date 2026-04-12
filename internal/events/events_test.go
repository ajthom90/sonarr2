package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fooEvent struct{ N int }
type barEvent struct{ S string }

func TestBusPublishDispatchesToMatchingSyncHandler(t *testing.T) {
	bus := NewBus(4)
	var got int
	SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
		got = e.N
		return nil
	})
	if err := bus.Publish(context.Background(), fooEvent{N: 42}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if got != 42 {
		t.Errorf("got = %d, want 42", got)
	}
}

func TestBusPublishIgnoresHandlersForOtherTypes(t *testing.T) {
	bus := NewBus(4)
	fooCalls := 0
	barCalls := 0
	SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
		fooCalls++
		return nil
	})
	SubscribeSync[barEvent](bus, func(ctx context.Context, e barEvent) error {
		barCalls++
		return nil
	})
	_ = bus.Publish(context.Background(), fooEvent{})
	if fooCalls != 1 || barCalls != 0 {
		t.Errorf("fooCalls=%d barCalls=%d, want 1/0", fooCalls, barCalls)
	}
}

func TestBusPublishPropagatesFirstSyncHandlerError(t *testing.T) {
	bus := NewBus(4)
	want := errors.New("boom")
	SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
		return want
	})
	// A second handler should also run even if the first errored, but the
	// returned error is the first one.
	called := false
	SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
		called = true
		return nil
	})
	err := bus.Publish(context.Background(), fooEvent{})
	if !errors.Is(err, want) {
		t.Errorf("err = %v, want %v", err, want)
	}
	if !called {
		t.Error("second handler must still run when first errors")
	}
}

func TestBusSubscribeOrderedRunsInOrder(t *testing.T) {
	bus := NewBus(4)
	var order []int
	var mu sync.Mutex
	record := func(n int) func(ctx context.Context, e fooEvent) error {
		return func(ctx context.Context, e fooEvent) error {
			mu.Lock()
			defer mu.Unlock()
			order = append(order, n)
			return nil
		}
	}
	// Register out of order.
	SubscribeOrdered[fooEvent](bus, 30, record(3))
	SubscribeOrdered[fooEvent](bus, 10, record(1))
	SubscribeOrdered[fooEvent](bus, 20, record(2))
	_ = bus.Publish(context.Background(), fooEvent{})
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Errorf("order = %v, want [1 2 3]", order)
	}
}

func TestBusSubscribeAsyncRunsInBackground(t *testing.T) {
	bus := NewBus(4)
	var got int64
	done := make(chan struct{})
	SubscribeAsync[fooEvent](bus, func(ctx context.Context, e fooEvent) {
		atomic.StoreInt64(&got, int64(e.N))
		close(done)
	})
	if err := bus.Publish(context.Background(), fooEvent{N: 7}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("async handler did not run within 1s")
	}
	if atomic.LoadInt64(&got) != 7 {
		t.Errorf("got = %d, want 7", got)
	}
}

func TestBusPublishWithNoHandlersReturnsNil(t *testing.T) {
	bus := NewBus(4)
	if err := bus.Publish(context.Background(), fooEvent{}); err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestBusConcurrentSubscribeAndPublishRaceFree(t *testing.T) {
	// Designed to exercise the mutex around the handler map under -race.
	bus := NewBus(16)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
				return nil
			})
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = bus.Publish(context.Background(), fooEvent{})
		}()
	}
	wg.Wait()
}

func TestNoopBusAcceptsAllOperations(t *testing.T) {
	bus := NewNoopBus()
	SubscribeSync[fooEvent](bus, func(ctx context.Context, e fooEvent) error {
		t.Error("noop bus should not invoke handlers")
		return nil
	})
	if err := bus.Publish(context.Background(), fooEvent{N: 1}); err != nil {
		t.Errorf("noop Publish error = %v", err)
	}
}
