// Package watcher watches a directory tree for file changes using fsnotify.
// It handles:
//   - Recursive watching of all subdirectories
//   - Dynamic addition of newly created directories
//   - Graceful removal of deleted directories
//   - OS inotify watch limit detection and warnings
package watcher

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Filterer determines whether a path should be ignored.
type Filterer interface {
	ShouldIgnore(path string) bool
}

// EventCallback is called for each relevant file system event.
type EventCallback func(path, event string)

// Watcher watches a directory tree and calls the callback on changes.
type Watcher struct {
	fw       *fsnotify.Watcher
	filter   Filterer
	callback EventCallback
	root     string

	mu      sync.Mutex
	watched map[string]struct{}
	done    chan struct{}
}

// New creates a Watcher rooted at root and starts watching immediately.
func New(root string, f Filterer, callback EventCallback) (*Watcher, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving root: %w", err)
	}

	checkWatchLimit()

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	w := &Watcher{
		fw:       fw,
		filter:   f,
		callback: callback,
		root:     root,
		watched:  make(map[string]struct{}),
		done:     make(chan struct{}),
	}

	if err := w.addTree(root); err != nil {
		fw.Close()
		return nil, fmt.Errorf("watching tree: %w", err)
	}

	go w.loop()
	return w, nil
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	close(w.done)
	return w.fw.Close()
}

// addTree recursively adds all subdirectories under root to the watcher.
func (w *Watcher) addTree(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			slog.Warn("walk error", "path", path, "err", err)
			return nil // skip unreadable paths
		}
		if !d.IsDir() {
			return nil
		}
		if w.filter.ShouldIgnore(path) {
			slog.Debug("ignoring directory", "path", path)
			return filepath.SkipDir
		}
		return w.addDir(path)
	})
}

func (w *Watcher) addDir(path string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, ok := w.watched[path]; ok {
		return nil
	}
	if err := w.fw.Add(path); err != nil {
		return fmt.Errorf("watching %s: %w", path, err)
	}
	w.watched[path] = struct{}{}
	slog.Debug("watching directory", "path", path)
	return nil
}

func (w *Watcher) removeDir(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.watched, path)
	// fsnotify auto-removes deleted paths; we just clean up our map.
}

// loop processes fsnotify events.
func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}
			w.handleEvent(event)

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}
			slog.Warn("watcher error", "err", err)
		}
	}
}

func (w *Watcher) handleEvent(event fsnotify.Event) {
	path := event.Name

	if w.filter.ShouldIgnore(path) {
		return
	}

	// If a new directory was created, watch it and its subtree
	if event.Has(fsnotify.Create) {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			slog.Debug("new directory detected, adding to watch", "path", path)
			if err := w.addTree(path); err != nil {
				slog.Warn("failed to watch new directory", "path", path, "err", err)
			}
			// Don't trigger a rebuild for directory creation
			return
		}
	}

	if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
		w.removeDir(path)
	}

	// Only trigger on meaningful file events
	if !event.Has(fsnotify.Write) &&
		!event.Has(fsnotify.Create) &&
		!event.Has(fsnotify.Remove) &&
		!event.Has(fsnotify.Rename) {
		return
	}

	// Skip non-source files (only watch .go files for Go projects, but keep
	// it general so the tool works for any language)
	if shouldSkipFile(path) {
		return
	}

	w.callback(path, event.Op.String())
}

// shouldSkipFile returns true for files that are noisy but irrelevant.
func shouldSkipFile(path string) bool {
	base := filepath.Base(path)
	// Editor swap/temp files
	if strings.HasSuffix(base, ".swp") ||
		strings.HasSuffix(base, ".swo") ||
		strings.HasSuffix(base, "~") ||
		strings.HasPrefix(base, ".#") ||
		strings.HasPrefix(base, "#") {
		return true
	}
	return false
}

// checkWatchLimit warns if the OS file-watch limit might be too low.
// On Linux this checks /proc/sys/fs/inotify/max_user_watches.
// On Windows there is no equivalent hard limit, so this is a no-op.
func checkWatchLimit() {
	if runtime.GOOS != "linux" {
		return
	}
	data, err := os.ReadFile("/proc/sys/fs/inotify/max_user_watches")
	if err != nil {
		return
	}
	limit, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return
	}
	if limit < 8192 {
		slog.Warn("inotify watch limit is low — you may hit 'too many open files' errors",
			"current_limit", limit,
			"suggestion", "run: echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p",
		)
	} else {
		slog.Debug("inotify watch limit", "limit", limit)
	}
}
