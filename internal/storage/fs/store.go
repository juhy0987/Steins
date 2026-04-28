// Package fs implements the manga and chapter repositories using the local
// filesystem as the source of truth. Intended for v0 — a future iteration
// will swap this out for a PostgreSQL-backed implementation behind the same
// interfaces.
//
// Filesystem layout:
//
//	<root>/<slug>/manga.json
//	<root>/<slug>/chapters/<number>/chapter.json   (optional)
//	<root>/<slug>/chapters/<number>/001.png        (page files)
//	<root>/<slug>/chapters/<number>/002.png
//
// Page filenames must be of the form `<digits>.<ext>` where ext is one of
// png/jpg/jpeg/webp. The digits determine the page index (1-based after sort).
//
// fs 패키지는 로컬 파일시스템을 진실 공급원으로 사용해 manga / chapter
// repository를 구현합니다. v0 전용이며, 후속 iteration에서 동일한 인터페이스
// 뒤에 PostgreSQL 기반 구현으로 교체됩니다.
package fs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"steins/internal/apperr"
	"steins/internal/chapter"
	"steins/internal/manga"
)

// Store is the in-memory snapshot of the on-disk catalog. It implements
// both manga.Repository and chapter.Repository.
type Store struct {
	root string

	mu       sync.RWMutex
	manga    map[string]*manga.Manga              // slug -> manga
	order    []string                             // slug list, original scan order
	chapters map[string]*chapter.Chapter          // chapter ID -> chapter
	bySlug   map[string][]*chapter.Chapter        // manga slug -> chapters
	pages    map[string][]*chapter.PageSource     // chapter ID -> ordered page sources
	pageByID map[string]map[int]*chapter.PageSource
}

// mangaFile mirrors the on-disk manga.json schema.
type mangaFile struct {
	Slug        string    `json:"slug"`
	Title       string    `json:"title"`
	AltTitles   []string  `json:"alt_titles,omitempty"`
	Description string    `json:"description"`
	CoverURL    string    `json:"cover_url,omitempty"`
	Authors     []string  `json:"authors,omitempty"`
	Genres      []string  `json:"genres,omitempty"`
	Status      string    `json:"status"`
	Language    string    `json:"language"`
	ReadingDir  string    `json:"reading_dir"`
	PublishedAt time.Time `json:"published_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// chapterFile mirrors the on-disk chapter.json schema (all fields optional).
type chapterFile struct {
	Title       string    `json:"title,omitempty"`
	Language    string    `json:"language,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// New scans the catalog rooted at `root` and returns a populated Store.
func New(root string) (*Store, error) {
	s := &Store{
		root:     root,
		manga:    make(map[string]*manga.Manga),
		chapters: make(map[string]*chapter.Chapter),
		bySlug:   make(map[string][]*chapter.Chapter),
		pages:    make(map[string][]*chapter.PageSource),
		pageByID: make(map[string]map[int]*chapter.PageSource),
	}
	if err := s.scan(); err != nil {
		return nil, err
	}
	return s, nil
}

// Root returns the catalog root path (used by tests).
func (s *Store) Root() string { return s.root }

func (s *Store) scan() error {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return fmt.Errorf("read catalog root %s: %w", s.root, err)
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if err := s.loadSeries(filepath.Join(s.root, e.Name()), e.Name()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadSeries(dir, slug string) error {
	metaPath := filepath.Join(dir, "manga.json")
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		// silently skip directories without manga.json — tolerate stray dirs
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", metaPath, err)
	}

	var mf mangaFile
	if err := json.Unmarshal(raw, &mf); err != nil {
		return fmt.Errorf("parse %s: %w", metaPath, err)
	}
	if mf.Slug == "" {
		mf.Slug = slug
	}

	m := &manga.Manga{
		ID:          mf.Slug,
		Slug:        mf.Slug,
		Title:       mf.Title,
		AltTitles:   mf.AltTitles,
		Description: mf.Description,
		CoverURL:    mf.CoverURL,
		Authors:     mf.Authors,
		Genres:      mf.Genres,
		Status:      mf.Status,
		Language:    mf.Language,
		ReadingDir:  mf.ReadingDir,
		PublishedAt: mf.PublishedAt,
		UpdatedAt:   mf.UpdatedAt,
	}

	s.manga[m.Slug] = m
	s.order = append(s.order, m.Slug)

	chDir := filepath.Join(dir, "chapters")
	chEntries, err := os.ReadDir(chDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", chDir, err)
	}

	chapters := make([]*chapter.Chapter, 0, len(chEntries))
	for _, ce := range chEntries {
		if !ce.IsDir() {
			continue
		}
		ch, sources, err := s.loadChapter(filepath.Join(chDir, ce.Name()), m, ce.Name())
		if err != nil {
			return err
		}
		if ch == nil {
			continue
		}
		chapters = append(chapters, ch)
		s.chapters[ch.ID] = ch
		s.pages[ch.ID] = sources
		idx := make(map[int]*chapter.PageSource, len(sources))
		for _, p := range sources {
			idx[p.Index] = p
		}
		s.pageByID[ch.ID] = idx
	}

	sortChapters(chapters)
	s.bySlug[m.Slug] = chapters
	return nil
}

func (s *Store) loadChapter(
	dir string, m *manga.Manga, number string,
) (*chapter.Chapter, []*chapter.PageSource, error) {
	pageEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", dir, err)
	}

	type rawPage struct {
		index int
		path  string
		mime  string
		size  int64
	}
	rawPages := make([]rawPage, 0, len(pageEntries))

	for _, pe := range pageEntries {
		if pe.IsDir() {
			continue
		}
		name := pe.Name()
		if name == "chapter.json" {
			continue
		}
		idx, mime, ok := parsePageFile(name)
		if !ok {
			continue
		}
		info, err := pe.Info()
		if err != nil {
			return nil, nil, fmt.Errorf("stat %s: %w", filepath.Join(dir, name), err)
		}
		rawPages = append(rawPages, rawPage{
			index: idx,
			path:  filepath.Join(dir, name),
			mime:  mime,
			size:  info.Size(),
		})
	}

	if len(rawPages) == 0 {
		// chapters with no pages are skipped
		return nil, nil, nil
	}

	sort.Slice(rawPages, func(i, j int) bool { return rawPages[i].index < rawPages[j].index })

	cf := chapterFile{}
	if raw, err := os.ReadFile(filepath.Join(dir, "chapter.json")); err == nil {
		_ = json.Unmarshal(raw, &cf)
	}

	chID := m.Slug + "__" + number
	ch := &chapter.Chapter{
		ID:          chID,
		MangaID:     m.ID,
		MangaSlug:   m.Slug,
		Number:      number,
		Title:       cf.Title,
		Language:    firstNonEmpty(cf.Language, m.Language),
		PageCount:   len(rawPages),
		PublishedAt: cf.PublishedAt,
	}

	sources := make([]*chapter.PageSource, 0, len(rawPages))
	for i, rp := range rawPages {
		sources = append(sources, &chapter.PageSource{
			ChapterID: chID,
			Index:     i + 1, // re-index 1..N regardless of filename gaps
			Path:      rp.path,
			MimeType:  rp.mime,
			ByteSize:  rp.size,
		})
	}

	return ch, sources, nil
}

// List implements manga.Repository.
func (s *Store) List(_ context.Context) ([]*manga.Manga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*manga.Manga, 0, len(s.order))
	for _, slug := range s.order {
		out = append(out, s.manga[slug])
	}
	return out, nil
}

// FindBySlug implements manga.Repository.
func (s *Store) FindBySlug(_ context.Context, slug string) (*manga.Manga, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	m, ok := s.manga[slug]
	if !ok {
		return nil, apperr.NewNotFoundError(apperr.CodeMangaNotFound, "manga", slug)
	}
	return m, nil
}

// ListByMangaSlug implements chapter.Repository.
func (s *Store) ListByMangaSlug(_ context.Context, slug string) ([]*chapter.Chapter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.manga[slug]; !ok {
		return nil, apperr.NewNotFoundError(apperr.CodeMangaNotFound, "manga", slug)
	}
	chapters, ok := s.bySlug[slug]
	if !ok {
		return []*chapter.Chapter{}, nil
	}
	out := make([]*chapter.Chapter, len(chapters))
	copy(out, chapters)
	return out, nil
}

// FindByID implements chapter.Repository.
func (s *Store) FindByID(_ context.Context, id string) (*chapter.Chapter, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ch, ok := s.chapters[id]
	if !ok {
		return nil, apperr.NewNotFoundError(apperr.CodeChapterNotFound, "chapter", id)
	}
	return ch, nil
}

// ListPageSources implements chapter.Repository.
func (s *Store) ListPageSources(_ context.Context, chapterID string) ([]*chapter.PageSource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.chapters[chapterID]; !ok {
		return nil, apperr.NewNotFoundError(apperr.CodeChapterNotFound, "chapter", chapterID)
	}
	src := s.pages[chapterID]
	out := make([]*chapter.PageSource, len(src))
	copy(out, src)
	return out, nil
}

// FindPageSource implements chapter.Repository.
func (s *Store) FindPageSource(
	_ context.Context, chapterID string, index int,
) (*chapter.PageSource, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.chapters[chapterID]; !ok {
		return nil, apperr.NewNotFoundError(apperr.CodeChapterNotFound, "chapter", chapterID)
	}
	idx, ok := s.pageByID[chapterID]
	if !ok {
		return nil, apperr.NewNotFoundError(apperr.CodePageNotFound, "page", strconv.Itoa(index))
	}
	p, ok := idx[index]
	if !ok {
		return nil, apperr.NewNotFoundError(apperr.CodePageNotFound, "page", strconv.Itoa(index))
	}
	return p, nil
}

func parsePageFile(name string) (int, string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	mime := mimeFromExt(ext)
	if mime == "" {
		return 0, "", false
	}
	stem := strings.TrimSuffix(name, ext)
	idx, err := strconv.Atoi(stem)
	if err != nil || idx <= 0 {
		return 0, "", false
	}
	return idx, mime, true
}

func mimeFromExt(ext string) string {
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	default:
		return ""
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// sortChapters orders chapters by ascending numeric value of `Number`,
// falling back to lexicographic when the value isn't a number.
func sortChapters(chs []*chapter.Chapter) {
	sort.SliceStable(chs, func(i, j int) bool {
		ai, aok := strconv.ParseFloat(chs[i].Number, 64)
		bi, bok := strconv.ParseFloat(chs[j].Number, 64)
		if aok == nil && bok == nil {
			return ai < bi
		}
		return chs[i].Number < chs[j].Number
	})
}
