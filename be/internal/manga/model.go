// Package manga defines the manga (series) domain — model, repository
// interface, and service.
//
// manga 패키지는 만화 series 도메인의 model, repository 인터페이스, service를
// 정의합니다.
package manga

import "time"

// Manga represents a manga series.
//
// Manga는 만화 시리즈를 나타냅니다.
type Manga struct {
	ID          string    `json:"id"`
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
