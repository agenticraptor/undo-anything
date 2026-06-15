package store

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Usage summarizes physical and logical disk usage for `ua status`.
type Usage struct {
	Snapshots   int   // number of timeline entries
	UniqueSnaps int   // distinct snapshot manifests on disk
	Objects     int   // number of stored blobs
	ObjectBytes int64 // physical bytes on disk (compressed)
	LogicalSize int64 // total size of files in the latest snapshot (uncompressed)
}

// Usage computes store statistics by walking the objects directory and reading
// the timeline.
func (s *Store) Usage() (Usage, error) {
	var u Usage
	entries, err := s.Index()
	if err != nil {
		return u, err
	}
	u.Snapshots = len(entries)
	seen := map[string]bool{}
	for _, e := range entries {
		if !seen[e.ID] {
			seen[e.ID] = true
			u.UniqueSnaps++
		}
	}
	walkErr := filepath.Walk(s.objectsPath(), func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		u.Objects++
		u.ObjectBytes += info.Size()
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return u, walkErr
	}
	if latest, err := s.Latest(); err == nil && latest != nil {
		u.LogicalSize = latest.Size
	}
	return u, nil
}

// GCResult reports what a garbage-collection pass removed.
type GCResult struct {
	SnapshotsRemoved int
	BlobsRemoved     int
	BytesReclaimed   int64
}

// GC removes every snapshot whose ID is not in keep, then deletes any blob no
// longer referenced by a surviving snapshot. It is safe to run at any time; an
// empty keep set is rejected by callers (prune always keeps at least one).
func (s *Store) GC(keep map[string]bool) (GCResult, error) {
	var res GCResult

	// 1. Collect hashes still referenced by kept snapshots.
	referenced := map[string]bool{}
	keptEntries := []IndexEntry{}
	entries, err := s.Index()
	if err != nil {
		return res, err
	}
	emittedManifest := map[string]bool{}
	for _, e := range entries {
		if keep[e.ID] {
			keptEntries = append(keptEntries, e)
			if !emittedManifest[e.ID] {
				emittedManifest[e.ID] = true
				snap, err := s.readSnapshotFile(e.ID)
				if err != nil {
					return res, err
				}
				for _, f := range snap.Files {
					referenced[f.Hash] = true
				}
			}
		}
	}

	// 2. Remove unreferenced snapshot manifests.
	manifestFiles, _ := filepath.Glob(filepath.Join(s.snapshotsPath(), "*.json"))
	for _, mf := range manifestFiles {
		id := trimManifestName(mf)
		if !keep[id] {
			if err := os.Remove(mf); err == nil {
				res.SnapshotsRemoved++
			}
		}
	}

	// 3. Rewrite the timeline to keep only surviving entries.
	if err := s.rewriteTimeline(keptEntries); err != nil {
		return res, err
	}

	// 4. Sweep unreferenced blobs.
	walkErr := filepath.Walk(s.objectsPath(), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		hash := filepath.Base(filepath.Dir(path)) + info.Name()
		if !referenced[hash] {
			size := info.Size()
			if err := os.Remove(path); err == nil {
				res.BlobsRemoved++
				res.BytesReclaimed += size
			}
		}
		return nil
	})
	if walkErr != nil && !os.IsNotExist(walkErr) {
		return res, walkErr
	}
	return res, nil
}

func (s *Store) readSnapshotFile(id string) (*Snapshot, error) {
	data, err := os.ReadFile(filepath.Join(s.snapshotsPath(), id+".json"))
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

func (s *Store) rewriteTimeline(entries []IndexEntry) error {
	var buf []byte
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf = append(buf, line...)
		buf = append(buf, '\n')
	}
	return s.atomicWrite(s.timelinePath(), buf)
}

func trimManifestName(path string) string {
	base := filepath.Base(path)
	return base[:len(base)-len(".json")]
}
