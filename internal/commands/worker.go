package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ajthom90/sonarr2/internal/events"
)

const (
	defaultLeaseDuration = 5 * time.Minute
	claimRetryInterval   = 200 * time.Millisecond
)

// WorkerPool runs N goroutines that claim, dispatch, and release commands in a
// loop. Panics inside handlers are caught and the command is transitioned to
// failed. Lease refresh runs every leaseDur/2. Graceful shutdown via Stop().
type WorkerPool struct {
	queue    Queue
	registry *Registry
	bus      events.Bus
	log      *slog.Logger
	workers  int
	leaseDur time.Duration
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewWorkerPool creates a WorkerPool with the default lease duration (5 min).
func NewWorkerPool(queue Queue, registry *Registry, bus events.Bus, log *slog.Logger, workers int) *WorkerPool {
	return &WorkerPool{
		queue:    queue,
		registry: registry,
		bus:      bus,
		log:      log,
		workers:  workers,
		leaseDur: defaultLeaseDuration,
	}
}

// Start launches N worker goroutines. It creates an internal context derived
// from ctx so that Stop() can cancel it independently of the caller's context.
func (wp *WorkerPool) Start(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	wp.cancel = cancel

	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go func(idx int) {
			defer wp.wg.Done()
			workerID := fmt.Sprintf("worker-%d-%d", os.Getpid(), idx)
			wp.run(innerCtx, workerID)
		}(i)
	}
}

// Stop cancels the internal context and waits for all workers to finish their
// current command. It is safe to call Stop multiple times.
func (wp *WorkerPool) Stop() {
	if wp.cancel != nil {
		wp.cancel()
	}
	wp.wg.Wait()
}

// run is the main loop for a single worker.
func (wp *WorkerPool) run(ctx context.Context, workerID string) {
	for {
		// Check for shutdown before attempting a claim.
		select {
		case <-ctx.Done():
			return
		default:
		}

		cmd, err := wp.queue.Claim(ctx, workerID, wp.leaseDur)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			wp.log.Error("worker: claim failed", "worker", workerID, "error", err)
			// Back off before retrying to avoid tight error loops.
			select {
			case <-ctx.Done():
				return
			case <-time.After(claimRetryInterval):
			}
			continue
		}

		if cmd == nil {
			// Nothing available — sleep and retry.
			select {
			case <-ctx.Done():
				return
			case <-time.After(claimRetryInterval):
			}
			continue
		}

		// We have a command; check context one more time before dispatching.
		select {
		case <-ctx.Done():
			// Best-effort: fail the command so it can be re-queued by the lease sweep.
			_ = wp.queue.Fail(ctx, cmd.ID, 0, "worker shutdown before dispatch", "shutdown")
			return
		default:
		}

		wp.dispatch(ctx, workerID, cmd)
	}
}

// dispatch executes a single command with panic recovery and lease refresh.
func (wp *WorkerPool) dispatch(ctx context.Context, workerID string, cmd *Command) {
	start := time.Now()

	// Publish CommandStarted.
	_ = wp.bus.Publish(ctx, CommandStarted{ID: cmd.ID, Name: cmd.Name})

	// Look up the handler.
	handler, err := wp.registry.Get(cmd.Name)
	if err != nil {
		durationMs := time.Since(start).Milliseconds()
		wp.log.Error("worker: no handler for command",
			"worker", workerID, "command", cmd.Name, "id", cmd.ID)
		_ = wp.queue.Fail(ctx, cmd.ID, durationMs, "no handler registered", "no handler registered")
		_ = wp.bus.Publish(ctx, CommandFailed{
			ID:        cmd.ID,
			Name:      cmd.Name,
			Exception: "no handler registered",
		})
		return
	}

	// Start the lease-refresh ticker.
	refreshInterval := wp.leaseDur / 2
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	// Run the refresh loop in a separate goroutine, stopping on tickerDone.
	tickerDone := make(chan struct{})
	defer close(tickerDone)

	go func() {
		for {
			select {
			case <-tickerDone:
				return
			case <-ticker.C:
				if err := wp.queue.RefreshLease(ctx, cmd.ID, wp.leaseDur); err != nil {
					wp.log.Warn("worker: lease refresh failed",
						"worker", workerID, "id", cmd.ID, "error", err)
				}
			}
		}
	}()

	// Execute the handler with panic recovery.
	var (
		handlerErr error
		panicMsg   string
		panicked   bool
	)

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				panicMsg = fmt.Sprintf("panic: %v", r)
			}
		}()
		handlerErr = handler.Handle(ctx, *cmd)
	}()

	durationMs := time.Since(start).Milliseconds()

	switch {
	case panicked:
		wp.log.Error("worker: handler panicked",
			"worker", workerID, "command", cmd.Name, "id", cmd.ID, "panic", panicMsg)
		_ = wp.queue.Fail(ctx, cmd.ID, durationMs, panicMsg, "panic")
		_ = wp.bus.Publish(ctx, CommandFailed{
			ID:        cmd.ID,
			Name:      cmd.Name,
			Exception: panicMsg,
		})

	case handlerErr != nil:
		wp.log.Error("worker: handler returned error",
			"worker", workerID, "command", cmd.Name, "id", cmd.ID, "error", handlerErr)
		_ = wp.queue.Fail(ctx, cmd.ID, durationMs, handlerErr.Error(), "handler error")
		_ = wp.bus.Publish(ctx, CommandFailed{
			ID:        cmd.ID,
			Name:      cmd.Name,
			Exception: handlerErr.Error(),
		})

	default:
		_ = wp.queue.Complete(ctx, cmd.ID, durationMs, nil, "ok")
		_ = wp.bus.Publish(ctx, CommandCompleted{
			ID:         cmd.ID,
			Name:       cmd.Name,
			DurationMs: durationMs,
		})
	}
}
