package snapshot

import (
	"sort"

	"github.com/agenticraptor/undo-anything/internal/store"
)

// ChangeKind classifies how a path changed between two snapshots.
type ChangeKind string

const (
	Added    ChangeKind = "added"
	Removed  ChangeKind = "removed"
	Modified ChangeKind = "modified"
)

// Change describes a single path-level difference between two snapshots.
type Change struct {
	Path    string
	Kind    ChangeKind
	OldHash string
	NewHash string
	OldSize int64
	NewSize int64
}

// Diff compares two snapshots (from -> to) and returns the path-level changes,
// sorted by path. Either snapshot may be nil, representing the empty tree.
func Diff(from, to *store.Snapshot) []Change {
	fromFiles := indexByPath(from)
	toFiles := indexByPath(to)

	var changes []Change
	for path, nf := range toFiles {
		of, ok := fromFiles[path]
		switch {
		case !ok:
			changes = append(changes, Change{Path: path, Kind: Added, NewHash: nf.Hash, NewSize: nf.Size})
		case of.Hash != nf.Hash:
			changes = append(changes, Change{
				Path: path, Kind: Modified,
				OldHash: of.Hash, NewHash: nf.Hash,
				OldSize: of.Size, NewSize: nf.Size,
			})
		}
	}
	for path, of := range fromFiles {
		if _, ok := toFiles[path]; !ok {
			changes = append(changes, Change{Path: path, Kind: Removed, OldHash: of.Hash, OldSize: of.Size})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}

func indexByPath(s *store.Snapshot) map[string]store.FileEntry {
	m := map[string]store.FileEntry{}
	if s == nil {
		return m
	}
	for _, f := range s.Files {
		m[f.Path] = f
	}
	return m
}
