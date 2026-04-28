// Package handler implements HTTP handlers for the public Steins API.
//
// handler 패키지는 Steins 공개 API의 HTTP 핸들러를 구현합니다.
package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	chi "github.com/go-chi/chi/v5"

	"steins/internal/apperr"
	"steins/internal/chapter"
	"steins/internal/manga"
	"steins/pkg/httpx"
)

// Handlers bundles the API handler dependencies.
type Handlers struct {
	Manga   *manga.Service
	Chapter *chapter.Service
}

// NewHandlers constructs the handler bundle.
func NewHandlers(m *manga.Service, c *chapter.Service) *Handlers {
	return &Handlers{Manga: m, Chapter: c}
}

// Health returns 200 OK for liveness probes.
func (h *Handlers) Health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// ListManga responds with the catalog of manga.
//
//	GET /api/v1/manga
func (h *Handlers) ListManga(w http.ResponseWriter, r *http.Request) {
	items, err := h.Manga.List(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteList(w, http.StatusOK, items, map[string]any{"count": len(items)})
}

// GetManga responds with a single manga by slug.
//
//	GET /api/v1/manga/{slug}
func (h *Handlers) GetManga(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		httpx.WriteError(w, apperr.NewValidationError(apperr.CodeValMissing, "slug is required", nil))
		return
	}
	m, err := h.Manga.GetBySlug(r.Context(), slug)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, m)
}

// ListChapters responds with the chapter list of a manga.
//
//	GET /api/v1/manga/{slug}/chapters
func (h *Handlers) ListChapters(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		httpx.WriteError(w, apperr.NewValidationError(apperr.CodeValMissing, "slug is required", nil))
		return
	}
	chapters, err := h.Chapter.ListByMangaSlug(r.Context(), slug)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteList(w, http.StatusOK, chapters, map[string]any{"count": len(chapters)})
}

// GetPageManifest responds with the chapter page manifest, including the
// SHA-256 checksum for each page so the client can verify image integrity
// after download.
//
//	GET /api/v1/chapters/{id}/pages
func (h *Handlers) GetPageManifest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, apperr.NewValidationError(apperr.CodeValMissing, "chapter id is required", nil))
		return
	}

	manifest, err := h.Chapter.GetPageManifest(r.Context(), id, pageURLBuilder)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, manifest)
}

// ServePageImage streams the raw page image and exposes the integrity
// checksum via the `Digest: sha-256=<base64>` response header (RFC 9530).
// The same value is also reachable through the page manifest, so the
// client may verify either way.
//
//	GET /api/v1/chapters/{id}/pages/{index}/image
func (h *Handlers) ServePageImage(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "id")
	indexParam := chi.URLParam(r, "index")
	if chapterID == "" || indexParam == "" {
		httpx.WriteError(w, apperr.NewValidationError(apperr.CodeValMissing, "chapter id and index are required", nil))
		return
	}

	idx, err := strconv.Atoi(indexParam)
	if err != nil || idx <= 0 {
		httpx.WriteError(w, apperr.NewValidationError(apperr.CodeValBadFormat, "index must be a positive integer", err))
		return
	}

	src, err := h.Chapter.FindPageSource(r.Context(), chapterID, idx)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	checksum, err := h.Chapter.Checksum(r.Context(), src)
	if err != nil {
		httpx.WriteError(w, apperr.NewStorageError(apperr.CodeStorageRead, "compute page checksum", err))
		return
	}

	f, err := os.Open(src.Path)
	if err != nil {
		httpx.WriteError(w, apperr.NewStorageError(apperr.CodeStorageRead, "open page file", err))
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", src.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(src.ByteSize, 10))
	// Format: `sha-256=<base64>` per RFC 9530. Our checksum is "sha-256:<base64>",
	// so split on the colon for the header form.
	if parts := strings.SplitN(string(checksum), ":", 2); len(parts) == 2 {
		w.Header().Set("Digest", fmt.Sprintf("%s=%s", parts[0], parts[1]))
		w.Header().Set("ETag", `"`+parts[1]+`"`)
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, f); err != nil {
		// Headers are already flushed; we can only log via the calling
		// middleware. Return without writing further.
		return
	}
}

// pageURLBuilder is the canonical URL pattern handed to the service so the
// service stays decoupled from chi routing.
func pageURLBuilder(chapterID string, pageIndex int) string {
	return fmt.Sprintf("/api/v1/chapters/%s/pages/%d/image", chapterID, pageIndex)
}
