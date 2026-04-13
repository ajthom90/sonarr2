package fswatcher

import (
	"sync"
	"time"
)

// debouncer coalesces repeated trigger calls for the same path into a single
// fire call. If Trigger is called while a timer is already pending for a path,
// the timer is reset to the full delay, so fire only runs after the last
// trigger in a burst.
type debouncer struct {
	mu      sync.Mutex
	pending map[string]*time.Timer
	fire    func(path string)
	delay   time.Duration
}

// newDebouncer constructs a debouncer that calls fire(path) after delay has
// elapsed since the last Trigger call for path.
func newDebouncer(delay time.Duration, fire func(path string)) *debouncer {
	return &debouncer{
		pending: make(map[string]*time.Timer),
		fire:    fire,
		delay:   delay,
	}
}

// Trigger schedules a fire for the given path. If a timer is already pending
// for this path, it is reset to the full delay.
func (d *debouncer) Trigger(path string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if t, ok := d.pending[path]; ok {
		t.Reset(d.delay)
		return
	}

	d.pending[path] = time.AfterFunc(d.delay, func() {
		d.mu.Lock()
		delete(d.pending, path)
		d.mu.Unlock()
		d.fire(path)
	})
}
