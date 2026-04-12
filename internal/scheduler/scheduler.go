package scheduler

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
)

const defaultInterval = 500 * time.Millisecond

// Scheduler reads scheduled_tasks on each tick and enqueues due commands.
// It also calls queue.SweepExpiredLeases on every tick to recover commands
// from crashed workers.
type Scheduler struct {
	taskStore TaskStore
	queue     commands.Queue
	log       *slog.Logger
	interval  time.Duration
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// New creates a Scheduler with the default 500 ms tick interval.
func New(taskStore TaskStore, queue commands.Queue, log *slog.Logger) *Scheduler {
	return &Scheduler{
		taskStore: taskStore,
		queue:     queue,
		log:       log,
		interval:  defaultInterval,
	}
}

// SetInterval overrides the tick interval. Intended for use in tests to speed
// up the scheduler. Must be called before Start.
func SetInterval(s *Scheduler, d time.Duration) {
	s.interval = d
}

// Start launches the tick loop in a background goroutine. The provided ctx is
// used only to detect external cancellation; the loop has its own internal
// cancel that is stopped by Stop.
func (s *Scheduler) Start(ctx context.Context) {
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run(loopCtx)
	}()
}

// Stop signals the tick loop to exit and waits for it to finish.
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	s.wg.Wait()
}

// run is the main tick loop executed in a goroutine.
func (s *Scheduler) run(ctx context.Context) {
	timer := time.NewTimer(s.interval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.tick(ctx)
			timer.Reset(s.interval)
		}
	}
}

// tick performs one scheduler cycle: enqueue due tasks then sweep leases.
func (s *Scheduler) tick(ctx context.Context) {
	tasks, err := s.taskStore.GetDueTasks(ctx)
	if err != nil {
		s.log.Error("scheduler: get due tasks", "err", err)
		// Still attempt the lease sweep even if task fetch failed.
	} else {
		for _, task := range tasks {
			s.processDueTask(ctx, task)
		}
	}

	if _, err := s.queue.SweepExpiredLeases(ctx); err != nil {
		s.log.Error("scheduler: sweep expired leases", "err", err)
	}
}

// processDueTask checks for duplicates and enqueues the command if none exist,
// then advances the task's next_execution.
func (s *Scheduler) processDueTask(ctx context.Context, task ScheduledTask) {
	dedupKey := task.TypeName

	// Skip if the same command is already queued or running.
	_, found, err := s.queue.FindDuplicate(ctx, dedupKey)
	if err != nil {
		s.log.Error("scheduler: find duplicate", "task", task.TypeName, "err", err)
		return
	}
	if found {
		s.log.Debug("scheduler: skipping due task — duplicate active", "task", task.TypeName)
		return
	}

	_, err = s.queue.Enqueue(ctx, task.TypeName, nil, commands.PriorityNormal, commands.TriggerScheduled, dedupKey)
	if err != nil {
		s.log.Error("scheduler: enqueue due task", "task", task.TypeName, "err", err)
		return
	}

	next := time.Now().Add(time.Duration(task.IntervalSecs) * time.Second)
	if err := s.taskStore.UpdateExecution(ctx, task.TypeName, next); err != nil {
		s.log.Error("scheduler: update execution", "task", task.TypeName, "err", err)
	}
}
