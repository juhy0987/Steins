// Package integrity provides image checksum computation used to defend
// against on-the-wire tampering of manga page images.
//
// Clients receive a per-page SHA-256 checksum in the page manifest and can
// re-compute the hash of the downloaded bytes to verify that what they
// received matches what the server intended to send.
//
// integrity 패키지는 만화 page 이미지의 전송 중 변조를 방지하기 위한 checksum
// 계산을 제공합니다. 클라이언트는 manifest의 SHA-256 checksum과 다운로드한
// 바이트의 해시를 비교해 무결성을 검증할 수 있습니다.
package integrity

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sync"
)

// Algorithm is the integrity algorithm prefix used in the checksum string.
const Algorithm = "sha-256"

// Checksum is a stringly-typed checksum in the form "sha-256:<base64>".
type Checksum string

// String returns the checksum value (Stringer for printing).
func (c Checksum) String() string { return string(c) }

// Calculator computes and caches SHA-256 checksums for files on disk.
// Cache entries are invalidated when the underlying file's modtime or size
// changes, so an on-disk update is detected automatically.
//
// Calculator는 디스크 파일의 SHA-256 checksum을 계산하고 캐시합니다.
// 파일의 modtime이나 크기가 변하면 캐시가 invalidate 되어 자동으로 재계산합니다.
type Calculator struct {
	mu    sync.RWMutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	checksum Checksum
	modtime  int64
	size     int64
}

// NewCalculator returns a Calculator with an empty cache.
func NewCalculator() *Calculator {
	return &Calculator{cache: make(map[string]cacheEntry)}
}

// OfFile returns the checksum for the file at the given path.
// Subsequent calls for the same path are served from the cache unless the
// file has been modified.
func (c *Calculator) OfFile(path string) (Checksum, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	mt := info.ModTime().UnixNano()
	sz := info.Size()

	c.mu.RLock()
	entry, ok := c.cache[path]
	c.mu.RUnlock()
	if ok && entry.modtime == mt && entry.size == sz {
		return entry.checksum, nil
	}

	checksum, err := computeFile(path)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.cache[path] = cacheEntry{checksum: checksum, modtime: mt, size: sz}
	c.mu.Unlock()

	return checksum, nil
}

// OfBytes returns the checksum of an in-memory byte slice (used by tests
// and callers that already hold the bytes).
func OfBytes(b []byte) Checksum {
	sum := sha256.Sum256(b)
	return Checksum(Algorithm + ":" + base64.StdEncoding.EncodeToString(sum[:]))
}

// OfReader streams the reader into a SHA-256 hasher and returns the digest.
func OfReader(r io.Reader) (Checksum, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", fmt.Errorf("hash stream: %w", err)
	}
	return Checksum(Algorithm + ":" + base64.StdEncoding.EncodeToString(h.Sum(nil))), nil
}

func computeFile(path string) (Checksum, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	return OfReader(f)
}
