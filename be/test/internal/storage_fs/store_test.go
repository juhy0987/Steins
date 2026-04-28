package fs_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"steins/internal/apperr"
	"steins/internal/storage/fs"
)

func TestStore_LoadsManga_AndChaptersOrdered(t *testing.T) {
	root := t.TempDir()
	seedSeries(t, root, "alpha", "Alpha", "ko", []seedChapter{
		{number: "0002", pages: 2},
		{number: "0001", pages: 3},
	})

	store, err := fs.New(root)
	require.NoError(t, err)

	ctx := context.Background()

	mangaList, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, mangaList, 1)
	assert.Equal(t, "alpha", mangaList[0].Slug)

	chapters, err := store.ListByMangaSlug(ctx, "alpha")
	require.NoError(t, err)
	require.Len(t, chapters, 2)
	assert.Equal(t, "0001", chapters[0].Number)
	assert.Equal(t, "0002", chapters[1].Number)
	assert.Equal(t, 3, chapters[0].PageCount)
}

func TestStore_FindBySlug_NotFound_ReturnsAppErr(t *testing.T) {
	store, err := fs.New(t.TempDir())
	require.NoError(t, err)

	_, err = store.FindBySlug(context.Background(), "missing")

	ae, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, apperr.CodeMangaNotFound, ae.Code)
}

func TestStore_PageSources_OrderedByIndex(t *testing.T) {
	root := t.TempDir()
	seedSeries(t, root, "alpha", "Alpha", "ko", []seedChapter{{number: "0001", pages: 4}})

	store, err := fs.New(root)
	require.NoError(t, err)

	sources, err := store.ListPageSources(context.Background(), "alpha__0001")
	require.NoError(t, err)
	require.Len(t, sources, 4)
	for i, src := range sources {
		assert.Equal(t, i+1, src.Index)
		assert.Equal(t, "image/png", src.MimeType)
	}
}

func TestStore_FindPageSource_BadIndex_ReturnsNotFound(t *testing.T) {
	root := t.TempDir()
	seedSeries(t, root, "alpha", "Alpha", "ko", []seedChapter{{number: "0001", pages: 2}})

	store, err := fs.New(root)
	require.NoError(t, err)

	_, err = store.FindPageSource(context.Background(), "alpha__0001", 99)
	ae, ok := apperr.As(err)
	require.True(t, ok)
	assert.Equal(t, apperr.CodePageNotFound, ae.Code)
}

// --- helpers ---

type seedChapter struct {
	number string
	pages  int
}

func seedSeries(t *testing.T, root, slug, title, lang string, chapters []seedChapter) {
	t.Helper()
	dir := filepath.Join(root, slug)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "chapters"), 0o755))

	meta := map[string]any{
		"slug":         slug,
		"title":        title,
		"description":  "test",
		"status":       "ongoing",
		"language":     lang,
		"reading_dir":  "ltr",
		"published_at": time.Now().UTC(),
		"updated_at":   time.Now().UTC(),
	}
	mb, _ := json.MarshalIndent(meta, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manga.json"), mb, 0o644))

	for _, c := range chapters {
		chDir := filepath.Join(dir, "chapters", c.number)
		require.NoError(t, os.MkdirAll(chDir, 0o755))
		for i := 1; i <= c.pages; i++ {
			// distinct bytes per page so checksums differ
			payload := []byte(slug + "-" + c.number + "-page-" + itoa(i))
			require.NoError(t, os.WriteFile(filepath.Join(chDir, padIndex(i)+".png"), payload, 0o644))
		}
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := []byte{}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	return string(digits)
}

func padIndex(i int) string {
	s := itoa(i)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
