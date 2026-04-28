package integrity_test

import (
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"steins/internal/integrity"
)

func TestOfBytes_KnownValue_MatchesManualHash(t *testing.T) {
	payload := []byte("hello manga")

	got := integrity.OfBytes(payload)

	sum := sha256.Sum256(payload)
	want := "sha-256:" + base64.StdEncoding.EncodeToString(sum[:])
	assert.Equal(t, want, got.String())
}

func TestCalculator_OfFile_MatchesBytesHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.bin")
	payload := []byte("steins page bytes")
	require.NoError(t, os.WriteFile(path, payload, 0o644))

	calc := integrity.NewCalculator()

	got, err := calc.OfFile(path)
	require.NoError(t, err)

	want := integrity.OfBytes(payload)
	assert.Equal(t, want, got)
}

func TestCalculator_OfFile_RecomputesAfterModification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.bin")
	require.NoError(t, os.WriteFile(path, []byte("v1"), 0o644))

	calc := integrity.NewCalculator()
	first, err := calc.OfFile(path)
	require.NoError(t, err)

	// Modify the file. Bump modtime to ensure cache invalidation even on
	// filesystems with coarse mtime resolution.
	require.NoError(t, os.WriteFile(path, []byte("v2-different-content"), 0o644))
	require.NoError(t, os.Chtimes(path, time.Now(), time.Now().Add(time.Second)))

	second, err := calc.OfFile(path)
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
	assert.Equal(t, integrity.OfBytes([]byte("v2-different-content")), second)
}

func TestCalculator_OfFile_MissingFile_ReturnsError(t *testing.T) {
	calc := integrity.NewCalculator()
	_, err := calc.OfFile(filepath.Join(t.TempDir(), "missing.bin"))
	assert.Error(t, err)
}
