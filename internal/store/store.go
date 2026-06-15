// Package store implements undo-anything's on-disk, content-addressed object
// store. It is the trust anchor of the whole tool: file contents are stored
// once per unique SHA-256 (deduplicated and compressed), and snapshots are
// immutable manifests that reference those blobs. Everything here is written
// atomically (temp file + rename) so an interrupted snapshot can never corrupt
// existing history.
package store

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// DirName is the per-folder store directory, analogous to ".git".
const DirName = ".undo"

const (
	objectsDir   = "objects"
	snapshotsDir = "snapshots"
	timelineLog  = "timeline.log"
	tmpDir       = "tmp"
)

// ErrNotInitialized is returned when no store is found for a path.
var ErrNotInitialized = errors.New("no undo-anything store found (run `ua init` first)")

// Store is a handle to a single folder's history.
type Store struct {
	// Root is the absolute path of the watched working directory.
	Root string
	// Dir is the absolute path of the .undo directory.
	Dir string
}

// Init creates a new store rooted at the given working directory. It is
// idempotent: re-initializing an existing store is a no-op that returns the
// existing handle.
func Init(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	s := &Store{Root: abs, Dir: filepath.Join(abs, DirName)}
	// The store holds unencrypted copies of your file history, so it is created
	// owner-only (0700). On an existing store we also tighten it, best effort.
	for _, d := range []string{"", objectsDir, snapshotsDir, tmpDir} {
		if err := os.MkdirAll(filepath.Join(s.Dir, d), 0o700); err != nil {
			return nil, err
		}
	}
	_ = os.Chmod(s.Dir, 0o700)
	// Ensure the timeline log exists so listing never fails on a fresh store.
	logPath := filepath.Join(s.Dir, timelineLog)
	if _, err := os.Stat(logPath); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(logPath, nil, 0o644); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Open returns a handle to the store at root, erroring if it is not
// initialized.
func Open(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(abs, DirName)
	if !isDir(dir) {
		return nil, ErrNotInitialized
	}
	return &Store{Root: abs, Dir: dir}, nil
}

// Find walks up from start looking for an existing store, mirroring how git
// discovers its repository root.
func Find(start string) (*Store, error) {
	abs, err := filepath.Abs(start)
	if err != nil {
		return nil, err
	}
	for {
		if isDir(filepath.Join(abs, DirName)) {
			return &Store{Root: abs, Dir: filepath.Join(abs, DirName)}, nil
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return nil, ErrNotInitialized
		}
		abs = parent
	}
}

func (s *Store) objectsPath() string   { return filepath.Join(s.Dir, objectsDir) }
func (s *Store) snapshotsPath() string { return filepath.Join(s.Dir, snapshotsDir) }
func (s *Store) timelinePath() string  { return filepath.Join(s.Dir, timelineLog) }
func (s *Store) tmpPath() string       { return filepath.Join(s.Dir, tmpDir) }

func isDir(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
