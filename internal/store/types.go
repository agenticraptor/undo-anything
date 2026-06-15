package store

import "time"

// Trigger describes what caused a snapshot to be taken.
const (
	TriggerManual     = "manual"      // `ua snapshot`
	TriggerWatch      = "watch"       // file change detected by the daemon
	TriggerPreRestore = "pre-restore" // automatic safety snapshot before a restore
	TriggerInit       = "init"        // initial snapshot at `ua init`
)

// FileEntry records a single tracked file within a snapshot. Hash is the
// SHA-256 (hex) of the file's uncompressed contents, which is also its key in
// the object store.
type FileEntry struct {
	Path    string    `json:"path"` // relative, slash-separated
	Hash    string    `json:"hash"`
	Size    int64     `json:"size"`
	Mode    uint32    `json:"mode"`
	ModTime time.Time `json:"modtime"`
}

// Stats summarizes a snapshot for quick display without rehydrating files.
type Stats struct {
	Files     int   `json:"files"`
	TotalSize int64 `json:"total_size"`
	NewBlobs  int   `json:"new_blobs"` // blobs written for the first time by this snapshot
	NewBytes  int64 `json:"new_bytes"` // physical bytes those new blobs occupy (compressed)
}

// Snapshot is a complete, content-addressed manifest of a watched tree at a
// point in time. The ID is the short hash of the canonical manifest, so two
// identical trees collapse to one snapshot.
type Snapshot struct {
	ID      string      `json:"id"`
	Time    time.Time   `json:"time"`
	Trigger string      `json:"trigger"`
	Label   string      `json:"label,omitempty"`
	Parent  string      `json:"parent,omitempty"`
	Files   []FileEntry `json:"files"`
	Stats   Stats       `json:"stats"`
}

// IndexEntry is the compact, append-only timeline record for a snapshot. The
// timeline log is a stream of these, newest last, used for fast listing.
type IndexEntry struct {
	ID       string    `json:"id"`
	Time     time.Time `json:"time"`
	Trigger  string    `json:"trigger"`
	Label    string    `json:"label,omitempty"`
	Parent   string    `json:"parent,omitempty"`
	Files    int       `json:"files"`
	Size     int64     `json:"size"`
	NewBlobs int       `json:"new_blobs"`
	NewBytes int64     `json:"new_bytes"`
}
