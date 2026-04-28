package chapter

import "context"

// Repository defines the data-access boundary for the chapter domain.
//
// Repository는 chapter 도메인의 데이터 접근 boundary입니다.
type Repository interface {
	ListByMangaSlug(ctx context.Context, slug string) ([]*Chapter, error)
	FindByID(ctx context.Context, id string) (*Chapter, error)
	ListPageSources(ctx context.Context, chapterID string) ([]*PageSource, error)
	FindPageSource(ctx context.Context, chapterID string, index int) (*PageSource, error)
}
