package chapter

import (
	"context"
	"fmt"

	"steins/internal/apperr"
	"steins/internal/integrity"
	"steins/internal/manga"
)

// PageURLBuilder produces the public URL for a given chapter page.
// Injected by the HTTP layer so the service does not depend on routing.
//
// PageURLBuilder는 chapter page의 공개 URL을 생성합니다. service가 라우팅에
// 의존하지 않도록 HTTP 레이어에서 주입합니다.
type PageURLBuilder func(chapterID string, pageIndex int) string

// Service is the application service for the chapter domain.
type Service struct {
	chapters    Repository
	mangaRepo   manga.Repository
	checksummer *integrity.Calculator
}

// NewService constructs a Service.
func NewService(chapters Repository, mangaRepo manga.Repository, c *integrity.Calculator) *Service {
	return &Service{chapters: chapters, mangaRepo: mangaRepo, checksummer: c}
}

// ListByMangaSlug returns the chapter list for a manga.
func (s *Service) ListByMangaSlug(ctx context.Context, slug string) ([]*Chapter, error) {
	return s.chapters.ListByMangaSlug(ctx, slug)
}

// GetByID returns a single chapter.
func (s *Service) GetByID(ctx context.Context, id string) (*Chapter, error) {
	return s.chapters.FindByID(ctx, id)
}

// GetPageManifest builds the page manifest, including the integrity checksum
// for each page. The provided urlBuilder produces the URL clients use to
// fetch the image bytes.
//
// GetPageManifest는 각 page의 무결성 checksum을 포함한 page manifest를 생성합니다.
// 전달된 urlBuilder가 클라이언트가 이미지 바이트를 가져올 URL을 생성합니다.
func (s *Service) GetPageManifest(
	ctx context.Context, chapterID string, urlBuilder PageURLBuilder,
) (*Manifest, error) {
	if urlBuilder == nil {
		return nil, apperr.NewInternalError("nil url builder", nil)
	}

	ch, err := s.chapters.FindByID(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	sources, err := s.chapters.ListPageSources(ctx, chapterID)
	if err != nil {
		return nil, err
	}

	m, err := s.mangaRepo.FindBySlug(ctx, ch.MangaSlug)
	if err != nil {
		return nil, fmt.Errorf("load manga for chapter %s: %w", ch.ID, err)
	}

	pages := make([]Page, 0, len(sources))
	for _, src := range sources {
		checksum, err := s.checksummer.OfFile(src.Path)
		if err != nil {
			return nil, apperr.NewStorageError(
				apperr.CodeStorageRead,
				fmt.Sprintf("checksum page %d of chapter %s", src.Index, ch.ID),
				err,
			)
		}
		pages = append(pages, Page{
			Index:    src.Index,
			URL:      urlBuilder(ch.ID, src.Index),
			ByteSize: src.ByteSize,
			MimeType: src.MimeType,
			Checksum: checksum.String(),
		})
	}

	readingDir := m.ReadingDir
	if readingDir == "" {
		readingDir = "ltr"
	}

	return &Manifest{
		ChapterID:  ch.ID,
		MangaID:    ch.MangaID,
		MangaSlug:  ch.MangaSlug,
		Number:     ch.Number,
		ReadingDir: readingDir,
		PageCount:  len(pages),
		Pages:      pages,
	}, nil
}

// FindPageSource returns the on-disk descriptor for a chapter page (used by
// the image-serving handler).
func (s *Service) FindPageSource(ctx context.Context, chapterID string, index int) (*PageSource, error) {
	return s.chapters.FindPageSource(ctx, chapterID, index)
}

// Checksum returns the cached SHA-256 checksum of a page's bytes.
func (s *Service) Checksum(_ context.Context, src *PageSource) (integrity.Checksum, error) {
	return s.checksummer.OfFile(src.Path)
}
