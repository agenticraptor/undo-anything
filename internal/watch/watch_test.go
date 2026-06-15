package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// TestWatchSnapshotsOnChange verifies the end-to-end watcher path: a file write
// produces a debounced snapshot that contains the new file.
func TestWatchSnapshotsOnChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping fs-watch integration test in -short mode")
	}
	s, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := ignore.NewWithDefaults(nil)

	events := make(chan Event, 8)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = Run(ctx, s, m, Options{Debounce: 120 * time.Millisecond}, func(ev Event) {
			events <- ev
		})
	}()

	// Give the watcher a moment to register its watches.
	time.Sleep(150 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(s.Root, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		if ev.Err != nil {
			t.Fatalf("watch event error: %v", ev.Err)
		}
		if !ev.Created || ev.Snapshot == nil {
			t.Fatalf("expected a created snapshot, got %+v", ev)
		}
		found := false
		for _, f := range ev.Snapshot.Files {
			if f.Path == "hello.txt" {
				found = true
			}
		}
		if !found {
			t.Errorf("snapshot did not contain hello.txt: %v", ev.Snapshot.Files)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a watch snapshot")
	}
}
