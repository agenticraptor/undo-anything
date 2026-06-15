package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateSnapshotRespectsIgnore(t *testing.T) {
	s := newStore(t)
	write(t, s.Root, "a.txt", "alpha")
	write(t, s.Root, "src/main.go", "package main")
	write(t, s.Root, "node_modules/dep/index.js", "junk") // ignored by default
	write(t, s.Root, "debug.log", "noise")                // .log ignored via extra

	m := ignore.NewWithDefaults([]string{"*.log"})
	snap, created, err := Create(s, m, Options{Trigger: store.TriggerInit})
	if err != nil || !created {
		t.Fatalf("Create: created=%v err=%v", created, err)
	}
	if snap.Stats.Files != 2 {
		t.Fatalf("expected 2 tracked files, got %d (%v)", snap.Stats.Files, paths(snap))
	}
	got := paths(snap)
	if got[0] != "a.txt" || got[1] != "src/main.go" {
		t.Errorf("tracked files = %v", got)
	}
}

func TestCreateIdempotent(t *testing.T) {
	s := newStore(t)
	write(t, s.Root, "a.txt", "alpha")
	m := ignore.NewWithDefaults(nil)

	first, created, err := Create(s, m, Options{Trigger: store.TriggerInit})
	if err != nil || !created {
		t.Fatalf("first create: %v", err)
	}
	second, created2, err := Create(s, m, Options{Trigger: store.TriggerWatch})
	if err != nil {
		t.Fatal(err)
	}
	if created2 {
		t.Error("unchanged tree should not create a new snapshot")
	}
	if first.ID != second.ID {
		t.Errorf("identical trees should share an ID: %s vs %s", first.ID, second.ID)
	}
}

func TestCreateDedupAndParent(t *testing.T) {
	s := newStore(t)
	write(t, s.Root, "a.txt", "alpha")
	m := ignore.NewWithDefaults(nil)
	first, _, _ := Create(s, m, Options{Trigger: store.TriggerInit})

	// Change a.txt and add b.txt.
	write(t, s.Root, "a.txt", "alpha v2")
	write(t, s.Root, "b.txt", "beta")
	second, created, err := Create(s, m, Options{Trigger: store.TriggerManual})
	if err != nil || !created {
		t.Fatalf("second create: created=%v err=%v", created, err)
	}
	if second.Parent != first.ID {
		t.Errorf("parent = %s, want %s", second.Parent, first.ID)
	}
	// Two blobs changed/added (a.txt v2 + b.txt); the original alpha blob stays.
	if second.Stats.NewBlobs != 2 {
		t.Errorf("NewBlobs = %d, want 2", second.Stats.NewBlobs)
	}

	changes := Diff(first, second)
	var added, modified int
	for _, c := range changes {
		switch c.Kind {
		case Added:
			added++
			if c.Path != "b.txt" {
				t.Errorf("unexpected added path %s", c.Path)
			}
		case Modified:
			modified++
			if c.Path != "a.txt" {
				t.Errorf("unexpected modified path %s", c.Path)
			}
		}
	}
	if added != 1 || modified != 1 {
		t.Errorf("diff = %d added, %d modified; want 1/1", added, modified)
	}
}

func TestDiffNilIsEmptyTree(t *testing.T) {
	s := newStore(t)
	write(t, s.Root, "only.txt", "x")
	m := ignore.NewWithDefaults(nil)
	snap, _, _ := Create(s, m, Options{Trigger: store.TriggerInit})

	changes := Diff(nil, snap)
	if len(changes) != 1 || changes[0].Kind != Added {
		t.Errorf("diff vs nil = %+v, want one Added", changes)
	}
}

func paths(s *store.Snapshot) []string {
	out := make([]string, 0, len(s.Files))
	for _, f := range s.Files {
		out = append(out, f.Path)
	}
	return out
}
