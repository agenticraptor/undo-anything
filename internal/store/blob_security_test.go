package store

import "testing"

// TestGetBlobRejectsOversized proves the decompression-bomb guard: a blob that
// expands beyond the supplied limit is rejected instead of being buffered whole.
func TestGetBlobRejectsOversized(t *testing.T) {
	s := newTestStore(t)
	// 1 KiB of compressible data; the exact size doesn't matter, only that it
	// exceeds the tiny limit we pass below.
	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 7)
	}
	hash, _, err := s.PutBlob(data)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.getBlob(hash, 100); err == nil {
		t.Error("expected getBlob to reject content larger than the limit")
	}
	// Within the limit it succeeds and round-trips exactly.
	got, err := s.getBlob(hash, int64(len(data)))
	if err != nil {
		t.Fatalf("getBlob within limit failed: %v", err)
	}
	if len(got) != len(data) {
		t.Errorf("round-trip length = %d, want %d", len(got), len(data))
	}
}

// TestGetBlobDefaultLimitOK confirms the public GetBlob (2 GiB ceiling) handles
// ordinary blobs without complaint.
func TestGetBlobDefaultLimitOK(t *testing.T) {
	s := newTestStore(t)
	hash, _, _ := s.PutBlob([]byte("hello world"))
	got, err := s.GetBlob(hash)
	if err != nil || string(got) != "hello world" {
		t.Fatalf("GetBlob = %q, err=%v", got, err)
	}
}
