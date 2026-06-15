package store

import (
	"os"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Init(t.TempDir())
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return s
}

func TestBlobRoundTripAndDedup(t *testing.T) {
	s := newTestStore(t)
	data := []byte("the quick brown fox")

	hash, written, err := s.PutBlob(data)
	if err != nil || !written {
		t.Fatalf("first PutBlob: written=%v err=%v", written, err)
	}
	if !s.HasBlob(hash) {
		t.Fatal("HasBlob should be true after PutBlob")
	}

	_, written2, err := s.PutBlob(data)
	if err != nil {
		t.Fatalf("second PutBlob: %v", err)
	}
	if written2 {
		t.Error("identical content should dedup (written=false)")
	}

	got, err := s.GetBlob(hash)
	if err != nil {
		t.Fatalf("GetBlob: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("GetBlob = %q, want %q", got, data)
	}

	otherHash, _, _ := s.PutBlob([]byte("different"))
	if otherHash == hash {
		t.Error("different content must produce different hash")
	}
}

func TestBlobCorruptionDetected(t *testing.T) {
	s := newTestStore(t)
	hash, _, _ := s.PutBlob([]byte("trust me"))
	// Overwrite the stored object with garbage.
	if err := os.WriteFile(s.blobPath(hash), []byte("not zlib at all"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetBlob(hash); err == nil {
		t.Error("GetBlob should fail on a corrupt blob")
	}
}

func snapWith(files []FileEntry, trigger string) *Snapshot {
	return &Snapshot{
		ID:      ComputeID(files),
		Time:    time.Now().UTC(),
		Trigger: trigger,
		Files:   files,
		Stats:   Stats{Files: len(files)},
	}
}

func TestSnapshotWriteReadIdempotent(t *testing.T) {
	s := newTestStore(t)
	files := []FileEntry{{Path: "a.txt", Hash: "aaa", Mode: 0o644}}
	snap := snapWith(files, TriggerManual)

	created, err := s.WriteSnapshot(snap)
	if err != nil || !created {
		t.Fatalf("WriteSnapshot: created=%v err=%v", created, err)
	}
	created2, err := s.WriteSnapshot(snap)
	if err != nil {
		t.Fatalf("WriteSnapshot 2: %v", err)
	}
	if created2 {
		t.Error("re-writing identical snapshot should be a no-op")
	}

	got, err := s.ReadSnapshot(snap.ID)
	if err != nil {
		t.Fatalf("ReadSnapshot: %v", err)
	}
	if got.ID != snap.ID || len(got.Files) != 1 {
		t.Errorf("ReadSnapshot mismatch: %+v", got)
	}

	// Prefix resolution.
	if _, err := s.ReadSnapshot(snap.ID[:6]); err != nil {
		t.Errorf("prefix resolve failed: %v", err)
	}
	if _, err := s.ResolveID("zzzzzz"); err == nil {
		t.Error("unknown prefix should error")
	}
}

func TestIndexOrderingAndLatest(t *testing.T) {
	s := newTestStore(t)
	first := snapWith([]FileEntry{{Path: "a", Hash: "1"}}, TriggerInit)
	second := snapWith([]FileEntry{{Path: "a", Hash: "2"}}, TriggerManual)
	if _, err := s.WriteSnapshot(first); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WriteSnapshot(second); err != nil {
		t.Fatal(err)
	}

	asc, _ := s.Index()
	if len(asc) != 2 || asc[0].ID != first.ID || asc[1].ID != second.ID {
		t.Fatalf("Index order wrong: %+v", asc)
	}
	desc, _ := s.IndexDesc()
	if desc[0].ID != second.ID {
		t.Errorf("IndexDesc[0] = %s, want newest %s", desc[0].ID, second.ID)
	}
	latest, _ := s.Latest()
	if latest == nil || latest.ID != second.ID {
		t.Errorf("Latest = %v, want %s", latest, second.ID)
	}
	if _, err := s.ResolveID("latest"); err != nil {
		t.Errorf("'latest' should resolve: %v", err)
	}
}

func TestGCRemovesUnreferenced(t *testing.T) {
	s := newTestStore(t)

	keepHash, _, _ := s.PutBlob([]byte("keep me"))
	dropHash, _, _ := s.PutBlob([]byte("drop me"))

	keepSnap := snapWith([]FileEntry{{Path: "keep.txt", Hash: keepHash}}, TriggerInit)
	dropSnap := snapWith([]FileEntry{{Path: "drop.txt", Hash: dropHash}}, TriggerManual)
	if _, err := s.WriteSnapshot(keepSnap); err != nil {
		t.Fatal(err)
	}
	if _, err := s.WriteSnapshot(dropSnap); err != nil {
		t.Fatal(err)
	}

	res, err := s.GC(map[string]bool{keepSnap.ID: true})
	if err != nil {
		t.Fatalf("GC: %v", err)
	}
	if res.SnapshotsRemoved != 1 {
		t.Errorf("SnapshotsRemoved = %d, want 1", res.SnapshotsRemoved)
	}
	if res.BlobsRemoved != 1 {
		t.Errorf("BlobsRemoved = %d, want 1", res.BlobsRemoved)
	}
	if !s.HasBlob(keepHash) {
		t.Error("referenced blob must survive GC")
	}
	if s.HasBlob(dropHash) {
		t.Error("unreferenced blob must be removed by GC")
	}
	// Timeline should now contain only the kept snapshot.
	idx, _ := s.Index()
	if len(idx) != 1 || idx[0].ID != keepSnap.ID {
		t.Errorf("timeline after GC = %+v", idx)
	}
}

func TestFindWalksUp(t *testing.T) {
	s := newTestStore(t)
	sub := s.Root + "/a/b/c"
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := Find(sub)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.Root != s.Root {
		t.Errorf("Find root = %s, want %s", found.Root, s.Root)
	}
}
