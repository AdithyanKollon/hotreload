// Package debouncer coalesces multiple rapid calls into a single callback.
// When Trigger is called, it waits for a quiet period (windowMs) before
// invoking the callback. Multiple triggers within the window reset the timer.
package debouncer

import (
	"sync"
	"time"
)

// Debouncer delays and deduplicates calls to a callback function.
type Debouncer struct {
	windowMs int
	callback func()

	mu      sync.Mutex
	timer   *time.Timer
	stopped bool
}

// New creates a Debouncer that calls callback after windowMs of inactivity.
func New(windowMs int, callback func()) *Debouncer {
	return &Debouncer{
		windowMs: windowMs,
		callback: callback,
	}
}

// Trigger schedules the callback. If called again before the window expires,
// the window resets.
func (d *Debouncer) Trigger() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.stopped {
		return
	}

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(time.Duration(d.windowMs)*time.Millisecond, func() {
		d.mu.Lock()
		stopped := d.stopped
		d.mu.Unlock()
		if !stopped {
			d.callback()
		}
	})
}

// Stop cancels any pending callback and prevents future callbacks.
func (d *Debouncer) Stop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stopped = true
	if d.timer != nil {
		d.timer.Stop()
	}
}
