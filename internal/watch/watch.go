// Package watch implements the recursive, debounced file watcher that powers
// the undo-anything daemon. fsnotify is not recursive, so this package manages
// watches for the whole subtree and snapshots the folder whenever activity
// settles.
package watch

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// Options configures the watcher.
type Options struct {
	// Debounce is how long activity must be quiet before a snapshot is taken.
	Debounce time.Duration
	// MaxFileSize bounds snapshotted files (0 = engine default).
	MaxFileSize int64
}

// Event is delivered to the caller after each settle. Exactly one of Snapshot
// or Err is meaningful; Created reports whether the snapshot was new (vs. an
// identical-state no-op).
type Event struct {
	Snapshot *store.Snapshot
	Created  bool
	Err      error
}

// Run watches the store's working tree until ctx is canceled, invoking onEvent
// after each debounced burst of filesystem activity. It returns when ctx is
// done or on a fatal watcher error.
func Run(ctx context.Context, s *store.Store, m *ignore.Matcher, opts Options, onEvent func(Event)) error {
	if m == nil {
		m = ignore.NewWithDefaults(nil)
	}
	if opts.Debounce <= 0 {
		opts.Debounce = 1500 * time.Millisecond
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	addTree(w, s.Root, s.Root, m)

	// Debounce timer is created stopped; we reset it on each relevant event.
	timer := time.NewTimer(time.Hour)
	if !timer.Stop() {
		<-timer.C
	}
	dirty := false

	snapshotNow := func() {
		if !dirty {
			return
		}
		dirty = false
		snap, created, err := snapshot.Create(s, m, snapshot.Options{
			Trigger:     store.TriggerWatch,
			MaxFileSize: opts.MaxFileSize,
		})
		onEvent(Event{Snapshot: snap, Created: created, Err: err})
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case event, ok := <-w.Events:
			if !ok {
				return nil
			}
			rel, relErr := filepath.Rel(s.Root, event.Name)
			if relErr != nil {
				continue
			}
			rel = filepath.ToSlash(rel)
			if isInternal(rel) {
				continue
			}
			// Newly created directories must be watched too.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Lstat(event.Name); err == nil && info.IsDir() {
					if !m.Match(rel, true) {
						addTree(w, event.Name, s.Root, m)
					}
				}
			}
			if m.Match(rel, false) {
				continue
			}
			dirty = true
			timer.Reset(opts.Debounce)

		case <-timer.C:
			snapshotNow()

		case _, ok := <-w.Errors:
			if !ok {
				return nil
			}
			// Transient watcher errors are non-fatal; keep going.
		}
	}
}

// addTree adds recursive watches for dir and all of its non-ignored
// subdirectories.
func addTree(w *fsnotify.Watcher, dir, root string, m *ignore.Matcher) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel != "." && (isInternal(rel) || m.Match(rel, true)) {
			return fs.SkipDir
		}
		_ = w.Add(path)
		return nil
	})
}

// isInternal reports whether a relative path is the store's own directory,
// which must never trigger snapshots (it would loop forever).
func isInternal(rel string) bool {
	return rel == store.DirName || strings.HasPrefix(rel, store.DirName+"/")
}
