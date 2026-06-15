package restore

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/store"
)

func TestSecureJoin(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	ok := []string{"a.txt", "sub/dir/b.txt", "a/../b.txt", "./c.txt"}
	for _, p := range ok {
		if _, err := secureJoin(root, p); err != nil {
			t.Errorf("secureJoin(%q) unexpectedly refused: %v", p, err)
		}
	}
	bad := []string{"../escape.txt", "../../etc/passwd", "a/../../escape.txt"}
	if runtime.GOOS != "windows" {
		bad = append(bad, "/etc/passwd")
	}
	for _, p := range bad {
		if _, err := secureJoin(root, p); err == nil {
			t.Errorf("secureJoin(%q) should have been refused", p)
		}
	}
}

// TestRestoreRefusesTraversal proves that restoring a tampered/foreign manifest
// whose entry path escapes the folder cannot write outside it.
func TestRestoreRefusesTraversal(t *testing.T) {
	s, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	hash, _, err := s.PutBlob([]byte("malicious"))
	if err != nil {
		t.Fatal(err)
	}
	// A manifest crafted to drop a file one level above the watched folder.
	evil := &store.Snapshot{
		ID:      "evil00000000",
		Trigger: store.TriggerManual,
		Files:   []store.FileEntry{{Path: "../pwned.txt", Hash: hash, Mode: 0o644}},
		Stats:   store.Stats{Files: 1},
	}
	if _, err := s.WriteSnapshot(evil); err != nil {
		t.Fatal(err)
	}

	m := ignore.NewWithDefaults(nil)
	if _, err := File(s, evil, "../pwned.txt", Options{Safety: false, Matcher: m}); err == nil {
		t.Fatal("expected traversal restore to be refused")
	}
	// The file must not exist anywhere above the root.
	outside := filepath.Join(filepath.Dir(s.Root), "pwned.txt")
	if _, err := os.Stat(outside); err == nil {
		t.Fatalf("SECURITY: traversal wrote outside the folder at %s", outside)
	}

	// A whole-tree restore of the same manifest must also refuse.
	if _, err := Snapshot(s, evil, Options{Safety: false, Matcher: m}); err == nil {
		t.Fatal("expected whole-tree traversal restore to be refused")
	}
}
