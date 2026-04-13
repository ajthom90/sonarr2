// Package fswatcher monitors filesystem root folders for changes and enqueues
// targeted ScanSeriesFolder commands when files appear or change. Each root
// folder gets its own fsnotify.Watcher. Events for the same path are coalesced
// within a debounce window (default 2 s) before a command is enqueued.
package fswatcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultDebounce = 2 * time.Second

// SeriesResolver maps a filesystem path to a series ID.
type SeriesResolver interface {
	ResolveSeriesID(path string) (int64, bool)
}

// CommandEnqueuer enqueues commands by name.
type CommandEnqueuer interface {
	Enqueue(ctx context.Context, name string, body []byte) error
}

// Watcher monitors root folders for filesystem changes and enqueues
// targeted ScanSeriesFolder commands when files change.
type Watcher struct {
	resolver SeriesResolver
	enqueuer CommandEnqueuer
	log      *slog.Logger
	debounce time.Duration
	mu       sync.Mutex
	watchers map[string]*fsnotify.Watcher // root path → watcher
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// New constructs a Watcher wired to the given resolver and enqueuer.
func New(resolver SeriesResolver, enqueuer CommandEnqueuer, log *slog.Logger) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &Watcher{
		resolver: resolver,
		enqueuer: enqueuer,
		log:      log,
		debounce: defaultDebounce,
		watchers: make(map[string]*fsnotify.Watcher),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// AddRoot starts watching rootPath for filesystem changes. A goroutine is
// started that reads fsnotify events, debounces them by path, and enqueues
// ScanSeriesFolder commands. Returns an error if the path cannot be watched.
func (w *Watcher) AddRoot(rootPath string) error {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("fswatcher: create watcher: %w", err)
	}
	if err := fw.Add(rootPath); err != nil {
		fw.Close()
		return fmt.Errorf("fswatcher: watch %q: %w", rootPath, err)
	}

	w.mu.Lock()
	w.watchers[rootPath] = fw
	w.mu.Unlock()

	deb := newDebouncer(w.debounce, func(path string) {
		seriesID, ok := w.resolver.ResolveSeriesID(path)
		if !ok {
			w.log.DebugContext(w.ctx, "fswatcher: no series for path", "path", path)
			return
		}
		body, err := json.Marshal(map[string]any{"seriesId": seriesID})
		if err != nil {
			w.log.ErrorContext(w.ctx, "fswatcher: marshal body", "error", err)
			return
		}
		if err := w.enqueuer.Enqueue(w.ctx, "ScanSeriesFolder", body); err != nil {
			w.log.WarnContext(w.ctx, "fswatcher: enqueue ScanSeriesFolder",
				"seriesId", seriesID, "error", err)
		}
	})

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		for {
			select {
			case event, ok := <-fw.Events:
				if !ok {
					return
				}
				if event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename) != 0 {
					deb.Trigger(event.Name)
				}
			case err, ok := <-fw.Errors:
				if !ok {
					return
				}
				w.log.WarnContext(w.ctx, "fswatcher: fsnotify error",
					"root", rootPath, "error", err)
			case <-w.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// RemoveRoot stops watching rootPath. If rootPath is not currently watched,
// this is a no-op.
func (w *Watcher) RemoveRoot(rootPath string) error {
	w.mu.Lock()
	fw, ok := w.watchers[rootPath]
	if ok {
		delete(w.watchers, rootPath)
	}
	w.mu.Unlock()

	if !ok {
		return nil
	}
	return fw.Close()
}

// Stop shuts down all watchers and waits for all goroutines to exit.
func (w *Watcher) Stop() {
	w.cancel()

	w.mu.Lock()
	watchers := make([]*fsnotify.Watcher, 0, len(w.watchers))
	for _, fw := range w.watchers {
		watchers = append(watchers, fw)
	}
	w.watchers = make(map[string]*fsnotify.Watcher)
	w.mu.Unlock()

	for _, fw := range watchers {
		fw.Close()
	}
	w.wg.Wait()
}
