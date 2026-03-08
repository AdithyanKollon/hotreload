// Package runner manages the lifecycle of the server process.
// Windows implementation uses taskkill /F /T to immediately kill process trees.
package runner

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	crashThreshold = 2 * time.Second
	maxBackoff     = 30 * time.Second
)

type Runner struct {
	cmd        string
	mu         sync.Mutex
	proc       *exec.Cmd
	startedAt  time.Time
	backoff    time.Duration
	crashCount int
	stopping   bool
}

func New(cmd string) *Runner { return &Runner{cmd: cmd} }

func (r *Runner) Restart(ctx context.Context) error {
	r.mu.Lock()
	r.stopping = false
	r.mu.Unlock()
	r.stop()
	return r.start(ctx)
}

func (r *Runner) Stop() {
	r.mu.Lock()
	r.stopping = true
	r.mu.Unlock()
	r.stop()
}

func (r *Runner) stop() {
	r.mu.Lock()
	proc := r.proc
	r.proc = nil
	r.mu.Unlock()

	if proc == nil || proc.Process == nil {
		return
	}

	pid := proc.Process.Pid
	slog.Debug("stopping server process", "pid", pid)

	// On Windows: immediately force-kill the entire process tree.
	// Go HTTP servers don't handle graceful shutdown signals on Windows,
	// so waiting for graceful exit just adds latency.
	killTree(pid, true)

	done := make(chan struct{})
	go func() {
		proc.Wait() //nolint:errcheck
		close(done)
	}()

	select {
	case <-done:
		slog.Debug("server process terminated")
	case <-time.After(2 * time.Second):
		slog.Warn("process did not terminate after kill, may be zombie")
	}
}

// killTree kills a process and all its children using taskkill.
func killTree(pid int, force bool) {
	args := []string{"/T", "/PID", fmt.Sprintf("%d", pid)}
	if force {
		args = append([]string{"/F"}, args...)
	}
	cmd := exec.Command("taskkill", args...)
	cmd.SysProcAttr = hiddenWindowAttr()
	if err := cmd.Run(); err != nil {
		slog.Debug("taskkill returned error (process may already be gone)", "err", err)
	}
}

func (r *Runner) start(ctx context.Context) error {
	r.mu.Lock()
	backoff := r.backoff
	r.mu.Unlock()

	if backoff > 0 {
		slog.Warn("crash loop detected, backing off", "backoff", backoff.Round(time.Millisecond))
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	parts := splitCommand(r.cmd)
	if len(parts) == 0 {
		return fmt.Errorf("empty exec command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.SysProcAttr = hiddenWindowAttr()
	cmd.Stdout = &passthroughWriter{w: os.Stdout}
	cmd.Stderr = &passthroughWriter{w: os.Stderr}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	now := time.Now()
	r.mu.Lock()
	r.proc = cmd
	r.startedAt = now
	r.mu.Unlock()

	slog.Info("server started", "pid", cmd.Process.Pid)
	go r.watch(cmd, now)
	return nil
}

func (r *Runner) watch(cmd *exec.Cmd, startedAt time.Time) {
	err := cmd.Wait()

	r.mu.Lock()
	if r.proc != cmd {
		r.mu.Unlock()
		return
	}
	r.proc = nil
	stopping := r.stopping
	elapsed := time.Since(startedAt)
	r.mu.Unlock()

	if stopping {
		return
	}

	if err != nil {
		slog.Error("server exited unexpectedly", "err", err, "uptime", elapsed.Round(time.Millisecond))
	} else {
		slog.Warn("server exited cleanly (unexpected)", "uptime", elapsed.Round(time.Millisecond))
	}

	if elapsed < crashThreshold {
		r.mu.Lock()
		r.crashCount++
		if r.backoff == 0 {
			r.backoff = 1 * time.Second
		} else {
			r.backoff *= 2
			if r.backoff > maxBackoff {
				r.backoff = maxBackoff
			}
		}
		slog.Warn("server crashed quickly", "crash_count", r.crashCount, "next_backoff", r.backoff.Round(time.Millisecond))
		r.mu.Unlock()
	} else {
		r.mu.Lock()
		r.crashCount = 0
		r.backoff = 0
		r.mu.Unlock()
	}
}

func splitCommand(s string) []string {
	var parts []string
	var cur strings.Builder
	inQuote := false
	quoteChar := rune(0)
	for _, r := range s {
		switch {
		case inQuote && r == quoteChar:
			inQuote = false
		case !inQuote && (r == '\'' || r == '"'):
			inQuote = true
			quoteChar = r
		case !inQuote && r == ' ':
			if cur.Len() > 0 {
				parts = append(parts, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		parts = append(parts, cur.String())
	}
	return parts
}

type passthroughWriter struct{ w io.Writer }

func (p *passthroughWriter) Write(b []byte) (int, error) { return p.w.Write(b) }
