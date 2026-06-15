package restore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
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

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "<missing>"
	}
	return string(b)
}

func exists(root, rel string) bool {
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// setup builds a store with two snapshots and returns the store, matcher, and
// the two snapshots (s1 = {a:v1,b:v1}, s2 = {a:v2,b:v1,c:new}).
func setup(t *testing.T) (*store.Store, *ignore.Matcher, *store.Snapshot, *store.Snapshot) {
	t.Helper()
	s, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := ignore.NewWithDefaults(nil)

	write(t, s.Root, "a.txt", "v1")
	write(t, s.Root, "b.txt", "v1")
	s1, _, err := snapshot.Create(s, m, snapshot.Options{Trigger: store.TriggerInit})
	if err != nil {
		t.Fatal(err)
	}

	write(t, s.Root, "a.txt", "v2")
	write(t, s.Root, "c.txt", "new")
	s2, _, err := snapshot.Create(s, m, snapshot.Options{Trigger: store.TriggerManual})
	if err != nil {
		t.Fatal(err)
	}
	return s, m, s1, s2
}

func TestRestoreFileTakesSafetyAndRevertsOne(t *testing.T) {
	s, m, s1, _ := setup(t)

	res, err := File(s, s1, "a.txt", Options{Safety: true, Matcher: m})
	if err != nil {
		t.Fatalf("File restore: %v", err)
	}
	if res.SafetyID == "" {
		t.Error("expected a safety snapshot id")
	}
	if got := read(t, s.Root, "a.txt"); got != "v1" {
		t.Errorf("a.txt = %q, want v1", got)
	}
	// Other files untouched.
	if got := read(t, s.Root, "c.txt"); got != "new" {
		t.Errorf("c.txt should be untouched, got %q", got)
	}
}

func TestRestoreTreeCleanRemovesExtraneous(t *testing.T) {
	s, m, s1, _ := setup(t)

	res, err := Snapshot(s, s1, Options{Safety: true, Clean: true, Matcher: m})
	if err != nil {
		t.Fatalf("tree restore: %v", err)
	}
	if got := read(t, s.Root, "a.txt"); got != "v1" {
		t.Errorf("a.txt = %q, want v1", got)
	}
	if exists(s.Root, "c.txt") {
		t.Error("c.txt should have been removed by --clean")
	}
	if len(res.Removed) != 1 || res.Removed[0] != "c.txt" {
		t.Errorf("Removed = %v, want [c.txt]", res.Removed)
	}
}

func TestRestoreIsItselfUndoable(t *testing.T) {
	s, m, s1, s2 := setup(t)

	// Restore back to s1 (clean), capturing the safety snapshot of s2 state.
	res, err := Snapshot(s, s1, Options{Safety: true, Clean: true, Matcher: m})
	if err != nil {
		t.Fatal(err)
	}
	if res.SafetyID == "" {
		t.Fatal("no safety snapshot captured")
	}
	// The safety snapshot must equal the pre-restore (s2) state.
	if res.SafetyID != s2.ID {
		t.Errorf("safety id = %s, want pre-restore state %s", res.SafetyID, s2.ID)
	}

	// Now undo the restore by restoring the safety snapshot.
	safety, err := s.ReadSnapshot(res.SafetyID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Snapshot(s, safety, Options{Safety: false, Clean: true, Matcher: m}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s.Root, "a.txt"); got != "v2" {
		t.Errorf("after undo a.txt = %q, want v2", got)
	}
	if !exists(s.Root, "c.txt") {
		t.Error("after undo c.txt should be back")
	}
}

func TestRestoreNoSafety(t *testing.T) {
	s, m, s1, _ := setup(t)
	res, err := File(s, s1, "a.txt", Options{Safety: false, Matcher: m})
	if err != nil {
		t.Fatal(err)
	}
	if res.SafetyID != "" {
		t.Errorf("expected no safety snapshot, got %s", res.SafetyID)
	}
}

func TestRestoreMissingFileErrors(t *testing.T) {
	s, m, s1, _ := setup(t)
	if _, err := File(s, s1, "does-not-exist.txt", Options{Safety: false, Matcher: m}); err == nil {
		t.Error("restoring a path not in the snapshot should error")
	}
}
