// Package builder runs a build command and calls a completion callback.
// Key properties:
//   - Only one build runs at a time.
//   - If Trigger is called while a build is running, the running build is
//     cancelled and a new one starts immediately after.
//   - The completion callback receives a bool indicating success.
package builder

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
)

// OnComplete is called when a build finishes.
type OnComplete func(success bool)

// Builder manages build lifecycle.
type Builder struct {
	cmd        string
	onComplete OnComplete

	mu        sync.Mutex
	cancelCur context.CancelFunc
	pending   bool
}

// New creates a Builder.
func New(cmd string, onComplete OnComplete) *Builder {
	return &Builder{cmd: cmd, onComplete: onComplete}
}

// Trigger requests a new build. If one is running it is cancelled first.
func (b *Builder) Trigger(ctx context.Context) {
	b.mu.Lock()
	if b.cancelCur != nil {
		slog.Debug("cancelling in-flight build for newer change")
		b.cancelCur()
		b.pending = true
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	go b.runBuild(ctx)
}

func (b *Builder) runBuild(ctx context.Context) {
	buildCtx, cancel := context.WithCancel(ctx)

	b.mu.Lock()
	b.cancelCur = cancel
	b.mu.Unlock()

	defer func() {
		cancel()
		b.mu.Lock()
		b.cancelCur = nil
		wasPending := b.pending
		b.pending = false
		b.mu.Unlock()

		if wasPending && ctx.Err() == nil {
			slog.Debug("starting queued build")
			go b.runBuild(ctx)
		}
	}()

	slog.Info("build started", "cmd", b.cmd)
	success := b.execute(buildCtx)
	if buildCtx.Err() != nil {
		slog.Debug("build was cancelled")
		return
	}
	b.onComplete(success)
}

func (b *Builder) execute(ctx context.Context) bool {
	parts := splitCommand(b.cmd)
	if len(parts) == 0 {
		slog.Error("empty build command")
		return false
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// On Windows: hide the console window for the build subprocess
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	cmd.Stdout = newPrefixWriter("│ BUILD │ ")
	cmd.Stderr = newPrefixWriter("│ BUILD │ ")

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return false
		}
		slog.Error("build failed", "err", err)
		return false
	}
	return true
}

// splitCommand splits a shell command string respecting quoted arguments.
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
