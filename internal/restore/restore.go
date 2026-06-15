// Package restore rehydrates files and whole trees from snapshots back onto the
// filesystem. Every restore is itself undoable: by default it first takes a
// "pre-restore" safety snapshot of the current state, so a mistaken restore is
// never destructive.
package restore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/agenticraptor/undo-anything/internal/ignore"
	"github.com/agenticraptor/undo-anything/internal/snapshot"
	"github.com/agenticraptor/undo-anything/internal/store"
)

// secureJoin resolves a snapshot-relative path against root and refuses
// anything that would escape it — an absolute path, a `..` traversal, or a
// component that resolves (via an existing symlink) outside root. This is the
// guard that makes restoring an untrusted or tampered .undo manifest safe: a
// crafted entry like "../../.ssh/authorized_keys" can never write outside the
// folder you ran `ua` in.
func secureJoin(root, rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing unsafe path %q (escapes the folder)", rel)
	}
	abs := filepath.Join(root, clean)
	rootClean := filepath.Clean(root)
	if abs != rootClean && !strings.HasPrefix(abs, rootClean+string(os.PathSeparator)) {
		return "", fmt.Errorf("refusing unsafe path %q (escapes the folder)", rel)
	}
	// Defense in depth: if an existing parent component is a symlink pointing
	// outside root, refuse before writing through it.
	if real, err := filepath.EvalSymlinks(filepath.Dir(abs)); err == nil {
		rootReal, rerr := filepath.EvalSymlinks(rootClean)
		if rerr != nil {
			rootReal = rootClean
		}
		if real != rootReal && !strings.HasPrefix(real, rootReal+string(os.PathSeparator)) {
			return "", fmt.Errorf("refusing path %q (parent directory resolves outside the folder)", rel)
		}
	}
	return abs, nil
}

// Options configures a restore operation.
type Options struct {
	// Safety takes a pre-restore snapshot first (recommended; default via New).
	Safety bool
	// Clean removes tracked files that did not exist in the target snapshot
	// (whole-tree restores only). Ignored/untracked files are never touched.
	Clean bool
	// Matcher identifies which working-tree files are "tracked" for cleaning
	// and for the safety snapshot.
	Matcher *ignore.Matcher
	// MaxFileSize bounds the safety snapshot (0 = engine default).
	MaxFileSize int64
}

// Result reports what a restore did.
type Result struct {
	SafetyID string   // ID of the auto safety snapshot ("" if skipped)
	Restored []string // files written
	Removed  []string // files removed (clean mode)
}

// Snapshot restores the entire working tree to match snap.
func Snapshot(s *store.Store, snap *store.Snapshot, opts Options) (Result, error) {
	var res Result
	if err := takeSafety(s, snap.ID, opts, &res); err != nil {
		return res, err
	}

	want := map[string]store.FileEntry{}
	for _, f := range snap.Files {
		want[f.Path] = f
		if err := writeFile(s, f); err != nil {
			return res, err
		}
		res.Restored = append(res.Restored, f.Path)
	}

	if opts.Clean {
		removed, err := cleanExtraneous(s, want, opts.Matcher)
		if err != nil {
			return res, err
		}
		res.Removed = removed
	}
	sort.Strings(res.Restored)
	sort.Strings(res.Removed)
	return res, nil
}

// File restores a single path from snap onto the working tree.
func File(s *store.Store, snap *store.Snapshot, relPath string, opts Options) (Result, error) {
	var res Result
	relPath = filepath.ToSlash(relPath)
	var target *store.FileEntry
	for i := range snap.Files {
		if snap.Files[i].Path == relPath {
			target = &snap.Files[i]
			break
		}
	}
	if target == nil {
		return res, fmt.Errorf("file %q is not present in snapshot %s", relPath, snap.ID)
	}
	if err := takeSafety(s, snap.ID, opts, &res); err != nil {
		return res, err
	}
	if err := writeFile(s, *target); err != nil {
		return res, err
	}
	res.Restored = append(res.Restored, target.Path)
	return res, nil
}

func takeSafety(s *store.Store, targetID string, opts Options, res *Result) error {
	if !opts.Safety {
		return nil
	}
	safety, _, err := snapshot.Create(s, opts.Matcher, snapshot.Options{
		Trigger:     store.TriggerPreRestore,
		Label:       fmt.Sprintf("before restore to %s", targetID),
		MaxFileSize: opts.MaxFileSize,
	})
	if err != nil {
		return fmt.Errorf("safety snapshot failed (nothing was changed): %w", err)
	}
	if safety != nil {
		res.SafetyID = safety.ID
	}
	return nil
}

// writeFile materializes one file entry from the object store atomically.
func writeFile(s *store.Store, f store.FileEntry) error {
	abs, err := secureJoin(s.Root, f.Path)
	if err != nil {
		return err
	}
	data, err := s.GetBlob(f.Hash)
	if err != nil {
		return fmt.Errorf("read blob for %s: %w", f.Path, err)
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(abs), ".ua-restore-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	mode := os.FileMode(f.Mode).Perm()
	if mode == 0 {
		mode = 0o644
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return err
	}
	if !f.ModTime.IsZero() {
		_ = os.Chtimes(abs, time.Now(), f.ModTime)
	}
	return nil
}

// cleanExtraneous removes tracked files not present in want. It only removes
// files the matcher considers tracked, so ignored paths (node_modules, build
// output, the store itself) are always safe.
func cleanExtraneous(s *store.Store, want map[string]store.FileEntry, m *ignore.Matcher) ([]string, error) {
	if m == nil {
		m = ignore.NewWithDefaults(nil)
	}
	var removed []string
	err := filepath.WalkDir(s.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
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
		if !d.Type().IsRegular() || m.Match(rel, false) {
			return nil
		}
		if _, keep := want[rel]; !keep {
			if err := os.Remove(path); err == nil {
				removed = append(removed, rel)
			}
		}
		return nil
	})
	return removed, err
}
