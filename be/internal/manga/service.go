package manga

import "context"

// Service is the application service for the manga domain.
type Service struct {
	repo Repository
}

// NewService constructs a Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns the catalog of manga.
func (s *Service) List(ctx context.Context) ([]*Manga, error) {
	return s.repo.List(ctx)
}

// GetBySlug returns a single manga by its slug.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*Manga, error) {
	return s.repo.FindBySlug(ctx, slug)
}
