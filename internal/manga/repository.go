package manga

import "context"

// Repository defines the data-access boundary for the manga domain.
//
// Repository는 manga 도메인의 데이터 접근 boundary입니다.
type Repository interface {
	List(ctx context.Context) ([]*Manga, error)
	FindBySlug(ctx context.Context, slug string) (*Manga, error)
}
