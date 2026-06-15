// Package snapshot turns a working directory into immutable, content-addressed
// snapshots in the store, and compares snapshots against each other. It is the
// bridge between the live filesystem and the object store.
package snapshot

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// DefaultMaxFileSize is the largest file snapshotted by default. Larger files
// are skipped to keep the store lean; raise it in config if you need to.
const DefaultMaxFileSize int64 = 50 << 20 // 50 MiB

// Options controls how a snapshot is created.
type Options struct {
	Trigger     string
	Label       string
	MaxFileSize int64 // 0 means DefaultMaxFileSize
}

// Create scans the store's working tree, writes any new blobs, and records a
// snapshot. It returns the snapshot and whether it was newly created (false
// means the tree was identical to an existing snapshot — a no-op).
func Create(s *store.Store, m *ignore.Matcher, opts Options) (*store.Snapshot, bool, error) {
	if m == nil {
		m = ignore.NewWithDefaults(nil)
	}
	maxSize := opts.MaxFileSize
	if maxSize <= 0 {
		maxSize = DefaultMaxFileSize
	}

	var (
		files    []store.FileEntry
		newBlobs int
		newBytes int64
		total    int64
	)

	walkErr := filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable paths rather than aborting the whole snapshot.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == s.Root {
			return nil
		}
		rel, relErr := filepath.Rel(s.Root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if m.Match(rel, true) {
				return fs.SkipDir
			}
			return nil
		}
		// Only regular files are tracked (no symlinks, devices, sockets).
		if !d.Type().IsRegular() {
			return nil
		}
		if m.Match(rel, false) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return nil
		}
		if info.Size() > maxSize {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		hash, written, putErr := s.PutBlob(data)
		if putErr != nil {
			return putErr
		}
		if written {
			newBlobs++
			newBytes += compressedSize(data)
		}
		total += info.Size()
		files = append(files, store.FileEntry{
			Path:    rel,
			Hash:    hash,
			Size:    info.Size(),
			Mode:    uint32(info.Mode().Perm()),
			ModTime: info.ModTime().UTC(),
		})
		return nil
	})
	if walkErr != nil {
		return nil, false, walkErr
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	var parent string
	if latest, err := s.Latest(); err == nil && latest != nil {
		parent = latest.ID
	}

	snap := &store.Snapshot{
		ID:      store.ComputeID(files),
		Time:    time.Now().UTC(),
		Trigger: opts.Trigger,
		Label:   opts.Label,
		Parent:  parent,
		Files:   files,
		Stats: store.Stats{
			Files:     len(files),
			TotalSize: total,
			NewBlobs:  newBlobs,
			NewBytes:  newBytes,
		},
	}
	// Don't record a parent that equals self (identical re-snapshot guard).
	if snap.ID == parent {
		return snap, false, nil
	}
	created, err := s.WriteSnapshot(snap)
	if err != nil {
		return nil, false, err
	}
	return snap, created, nil
}

// compressedSize reports how many bytes a blob occupies after zlib compression,
// matching what the store actually writes. Used only for "new bytes" stats.
func compressedSize(data []byte) int64 {
	return int64(store.CompressedLen(data))
}
