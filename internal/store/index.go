package store

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ComputeID derives a snapshot's stable short ID from its file manifest. Two
// snapshots with identical file sets (path+hash+mode) collapse to one ID, which
// is what makes "snapshot on every save" cheap and idempotent.
func ComputeID(files []FileEntry) string {
	sorted := make([]FileEntry, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })
	h := sha256.New()
	for _, f := range sorted {
		fmt.Fprintf(h, "%s\x00%s\x00%o\x00", f.Path, f.Hash, f.Mode)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// WriteSnapshot persists a snapshot manifest and appends it to the timeline.
// It returns created=false if a snapshot with the same ID already exists (an
// identical-state no-op), in which case nothing is written.
func (s *Store) WriteSnapshot(snap *Snapshot) (created bool, err error) {
	path := filepath.Join(s.snapshotsPath(), snap.ID+".json")
	if _, statErr := os.Stat(path); statErr == nil {
		return false, nil
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(s.snapshotsPath(), 0o755); err != nil {
		return false, err
	}
	if err := s.atomicWrite(path, data); err != nil {
		return false, err
	}
	entry := IndexEntry{
		ID:       snap.ID,
		Time:     snap.Time,
		Trigger:  snap.Trigger,
		Label:    snap.Label,
		Parent:   snap.Parent,
		Files:    snap.Stats.Files,
		Size:     snap.Stats.TotalSize,
		NewBlobs: snap.Stats.NewBlobs,
		NewBytes: snap.Stats.NewBytes,
	}
	if err := s.appendIndex(entry); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) appendIndex(e IndexEntry) error {
	line, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.timelinePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

// Index returns all timeline entries in chronological order (oldest first).
func (s *Store) Index() ([]IndexEntry, error) {
	f, err := os.Open(s.timelinePath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var entries []IndexEntry
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e IndexEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("corrupt timeline entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}

// IndexDesc returns timeline entries newest first.
func (s *Store) IndexDesc() ([]IndexEntry, error) {
	entries, err := s.Index()
	if err != nil {
		return nil, err
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// Latest returns the most recent timeline entry, or nil if the store is empty.
func (s *Store) Latest() (*IndexEntry, error) {
	entries, err := s.Index()
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	last := entries[len(entries)-1]
	return &last, nil
}

// ResolveID expands a unique snapshot ID prefix to its full ID. The literal
// alias "latest" resolves to the newest snapshot.
func (s *Store) ResolveID(prefix string) (string, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return "", fmt.Errorf("empty snapshot id")
	}
	entries, err := s.Index()
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return "", fmt.Errorf("no snapshots yet")
	}
	if prefix == "latest" {
		return entries[len(entries)-1].ID, nil
	}
	var matches []string
	seen := map[string]bool{}
	for _, e := range entries {
		if strings.HasPrefix(e.ID, prefix) && !seen[e.ID] {
			matches = append(matches, e.ID)
			seen[e.ID] = true
		}
	}
	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no snapshot matches %q", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous snapshot id %q (matches %d snapshots)", prefix, len(matches))
	}
}

// ReadSnapshot loads a full snapshot manifest by ID or unique prefix.
func (s *Store) ReadSnapshot(id string) (*Snapshot, error) {
	full, err := s.ResolveID(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(s.snapshotsPath(), full+".json"))
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("corrupt snapshot %s: %w", full, err)
	}
	return &snap, nil
}
