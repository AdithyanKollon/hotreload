package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/AdithyanKollon/hotreload/builder"
	"github.com/AdithyanKollon/hotreload/debouncer"
	"github.com/AdithyanKollon/hotreload/filter"
	"github.com/AdithyanKollon/hotreload/runner"
	"github.com/AdithyanKollon/hotreload/watcher"
)

const version = "1.0.0"

func main() {
	var (
		root     = flag.String("root", ".", "Directory to watch for file changes")
		build    = flag.String("build", "", "Command to build the project (required)")
		exec     = flag.String("exec", "", "Command to run after a successful build (required)")
		logLevel = flag.String("log-level", "info", "Log level: debug, info, warn, error")
		ver      = flag.Bool("version", false, "Print version and exit")
	)
	flag.Parse()

	if *ver {
		fmt.Printf("hotreload version %s\n", version)
		os.Exit(0)
	}

	if *build == "" || *exec == "" {
		fmt.Fprintf(os.Stderr, "hotreload: --build and --exec are required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Configure structured logging
	level := slog.LevelInfo
	switch *logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	slog.Info("hotreload starting",
		"version", version,
		"root", *root,
		"build", *build,
		"exec", *exec,
	)

	// Top-level context cancelled on SIGINT/SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, *root, *build, *exec); err != nil {
		slog.Error("hotreload exited with error", "err", err)
		os.Exit(1)
	}
	slog.Info("hotreload stopped")
}

func run(ctx context.Context, root, buildCmd, execCmd string) error {
	// File filter — ignore noisy paths
	f := filter.New(filter.DefaultIgnorePatterns...)

	// Process runner
	r := runner.New(execCmd)

	// Builder — cancels in-flight builds when a new change arrives
	b := builder.New(buildCmd, func(success bool) {
		if !success {
			slog.Warn("build failed, keeping previous server running")
			return
		}
		slog.Info("build succeeded, restarting server")
		if err := r.Restart(ctx); err != nil {
			slog.Error("failed to restart server", "err", err)
		}
	})

	// Debouncer — coalesces rapid file events into a single trigger
	deb := debouncer.New(300, func() {
		slog.Info("change detected, triggering build")
		b.Trigger(ctx)
	})
	defer deb.Stop()

	// File watcher
	w, err := watcher.New(root, f, func(path, event string) {
		slog.Debug("file event", "path", path, "event", event)
		deb.Trigger()
	})
	if err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}
	defer w.Close()

	// Trigger the first build immediately without waiting for a file change
	slog.Info("triggering initial build")
	b.Trigger(ctx)

	// Block until context is cancelled (SIGINT/SIGTERM)
	<-ctx.Done()

	slog.Info("shutting down...")
	r.Stop()
	return nil
}
