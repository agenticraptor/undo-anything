package store

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// HashBytes returns the hex SHA-256 of data. This is the object key.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// CompressedLen returns the number of bytes data occupies after zlib
// compression — i.e. its physical footprint in the object store.
func CompressedLen(data []byte) int {
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	_, _ = zw.Write(data)
	_ = zw.Close()
	return buf.Len()
}

func (s *Store) blobPath(hash string) string {
	// Shard by the first two hex chars to avoid huge flat directories.
	return filepath.Join(s.objectsPath(), hash[:2], hash[2:])
}

// HasBlob reports whether a blob with the given hash already exists.
func (s *Store) HasBlob(hash string) bool {
	if len(hash) < 3 {
		return false
	}
	_, err := os.Stat(s.blobPath(hash))
	return err == nil
}

// PutBlob stores data, returning its hash and whether it was newly written
// (false means it was already present — a dedup hit). Storage is atomic.
func (s *Store) PutBlob(data []byte) (hash string, written bool, err error) {
	hash = HashBytes(data)
	dst := s.blobPath(hash)
	if _, statErr := os.Stat(dst); statErr == nil {
		return hash, false, nil // already stored
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", false, err
	}
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return "", false, err
	}
	if err := zw.Close(); err != nil {
		return "", false, err
	}
	if err := s.atomicWrite(dst, buf.Bytes()); err != nil {
		return "", false, err
	}
	return hash, true, nil
}

// MaxBlobBytes is a hard ceiling on how many bytes a single blob may
// decompress to. It guards against a "decompression bomb" — a tiny crafted
// object in an untrusted .undo store that would otherwise expand to gigabytes
// and exhaust memory. Legitimate blobs never approach this (they are bounded by
// the much smaller max_file_size setting at snapshot time).
const MaxBlobBytes int64 = 2 << 30 // 2 GiB

// GetBlob returns the decompressed contents of the blob with the given hash.
func (s *Store) GetBlob(hash string) ([]byte, error) {
	return s.getBlob(hash, MaxBlobBytes)
}

func (s *Store) getBlob(hash string, limit int64) ([]byte, error) {
	f, err := os.Open(s.blobPath(hash))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	zr, err := zlib.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("decompress blob %s: %w", short(hash), err)
	}
	defer zr.Close()
	// Read at most limit+1 bytes so we can detect (and reject) anything that
	// exceeds the ceiling without buffering it all.
	data, err := io.ReadAll(io.LimitReader(zr, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("blob %s exceeds the %d-byte safety limit (possible decompression bomb)", short(hash), limit)
	}
	// Verify integrity: the decompressed content must hash back to its key.
	if got := HashBytes(data); got != hash {
		return nil, fmt.Errorf("blob %s is corrupt (content hash %s)", short(hash), short(got))
	}
	return data, nil
}

// atomicWrite writes data to a temp file in the store's tmp dir, then renames
// it into place so readers never observe a partial file.
func (s *Store) atomicWrite(dst string, data []byte) error {
	if err := os.MkdirAll(s.tmpPath(), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.tmpPath(), "obj-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if the rename succeeded
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
	return os.Rename(tmpName, dst)
}

func short(hash string) string {
	if len(hash) > 8 {
		return hash[:8]
	}
	return hash
}
