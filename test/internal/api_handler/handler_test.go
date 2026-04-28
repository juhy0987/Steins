package handler_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"steins/internal/api"
	"steins/internal/api/handler"
	"steins/internal/chapter"
	"steins/internal/integrity"
	"steins/internal/manga"
	"steins/internal/storage/fs"
	"steins/pkg/logger"
)

// Verifies the end-to-end integrity contract: the manifest checksum, the
// `Digest` response header, and the SHA-256 of the served bytes must all
// agree. A mismatch here would mean a client cannot detect tampering.
func TestServePageImage_DigestAndManifestMatchBodyHash(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	manifest := fetchManifest(t, srv, "alpha__0001")
	require.Len(t, manifest.Pages, 2)

	for _, page := range manifest.Pages {
		resp, err := http.Get(srv.URL + page.URL)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		// Body hash equals manifest checksum.
		sum := sha256.Sum256(body)
		bodyChecksum := "sha-256:" + base64.StdEncoding.EncodeToString(sum[:])
		assert.Equal(t, page.Checksum, bodyChecksum, "body hash must match manifest checksum")

		// Digest header equals manifest checksum (with `=` separator instead of `:`).
		digest := resp.Header.Get("Digest")
		expected := "sha-256=" + base64.StdEncoding.EncodeToString(sum[:])
		assert.Equal(t, expected, digest, "Digest header must match body hash")
	}
}

func TestListManga_ReturnsSeededCatalog(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/manga")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Data []manga.Manga `json:"data"`
		Meta struct {
			Count int `json:"count"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.Len(t, body.Data, 1)
	assert.Equal(t, "alpha", body.Data[0].Slug)
	assert.Equal(t, 1, body.Meta.Count)
}

func TestGetManga_NotFound_Returns404(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/manga/missing")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestServePageImage_BadIndex_Returns400(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/chapters/alpha__0001/pages/abc/image")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- helpers ---

type testManifest struct {
	Data struct {
		ChapterID string         `json:"chapter_id"`
		Pages     []chapter.Page `json:"pages"`
	} `json:"data"`
}

func (m testManifest) Pages() []chapter.Page { return m.Data.Pages }

func fetchManifest(t *testing.T, srv *httptest.Server, chapterID string) struct {
	ChapterID string
	Pages     []chapter.Page
} {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/v1/chapters/" + chapterID + "/pages")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body testManifest
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return struct {
		ChapterID string
		Pages     []chapter.Page
	}{ChapterID: body.Data.ChapterID, Pages: body.Data.Pages}
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	root := t.TempDir()
	seedSimpleSeries(t, root)

	store, err := fs.New(root)
	require.NoError(t, err)

	mangaSvc := manga.NewService(store)
	chapterSvc := chapter.NewService(store, store, integrity.NewCalculator())
	handlers := handler.NewHandlers(mangaSvc, chapterSvc)

	router := api.NewRouter(api.Config{}, logger.New(false), handlers)
	return httptest.NewServer(router)
}

func seedSimpleSeries(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "alpha")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "chapters", "0001"), 0o755))

	meta := map[string]any{
		"slug":        "alpha",
		"title":       "Alpha",
		"status":      "ongoing",
		"language":    "ko",
		"reading_dir": "rtl",
		"published_at": time.Now().UTC(),
		"updated_at":   time.Now().UTC(),
	}
	mb, _ := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manga.json"), mb, 0o644))

	pages := [][]byte{
		[]byte("page-one-bytes"),
		[]byte("page-two-bytes-different"),
	}
	for i, payload := range pages {
		name := []byte{'0', '0', byte('1' + i), '.', 'p', 'n', 'g'}
		require.NoError(t, os.WriteFile(filepath.Join(dir, "chapters", "0001", string(name)), payload, 0o644))
	}
}
