package debouncer_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdithyanKollon/hotreload/debouncer"
)

func TestDebouncerFiresOnce(t *testing.T) {
	var count atomic.Int32
	d := debouncer.New(50, func() { count.Add(1) })
	defer d.Stop()

	// Rapid triggers — should only fire once
	for i := 0; i < 10; i++ {
		d.Trigger()
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	if got := count.Load(); got != 1 {
		t.Errorf("expected callback to fire once, got %d", got)
	}
}

func TestDebouncerFiresAfterQuietPeriod(t *testing.T) {
	var count atomic.Int32
	d := debouncer.New(50, func() { count.Add(1) })
	defer d.Stop()

	d.Trigger()
	time.Sleep(200 * time.Millisecond) // wait past window

	d.Trigger()
	time.Sleep(200 * time.Millisecond) // wait past window

	if got := count.Load(); got != 2 {
		t.Errorf("expected 2 callbacks, got %d", got)
	}
}

func TestDebouncerStopPreventsCallback(t *testing.T) {
	var count atomic.Int32
	d := debouncer.New(100, func() { count.Add(1) })

	d.Trigger()
	d.Stop() // stop before window expires

	time.Sleep(300 * time.Millisecond)

	if got := count.Load(); got != 0 {
		t.Errorf("expected 0 callbacks after Stop, got %d", got)
	}
}

func TestDebouncerResetOnRetrigger(t *testing.T) {
	var count atomic.Int32
	start := time.Now()
	d := debouncer.New(100, func() {
		count.Add(1)
	})
	defer d.Stop()

	// Trigger every 50ms for 200ms total — window keeps resetting
	for i := 0; i < 4; i++ {
		d.Trigger()
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(300 * time.Millisecond)

	elapsed := time.Since(start)
	_ = elapsed

	if got := count.Load(); got != 1 {
		t.Errorf("expected exactly 1 callback, got %d", got)
	}
}
