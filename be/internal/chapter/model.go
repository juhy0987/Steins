// Package chapter defines the chapter and page domain — models, repository
// interface, and service.
//
// chapter 패키지는 chapter와 page 도메인의 model, repository 인터페이스, service를
// 정의합니다.
package chapter

import "time"

// Chapter represents a single chapter of a manga.
//
// Chapter는 만화의 단일 chapter를 나타냅니다.
type Chapter struct {
	ID          string    `json:"id"`
	MangaID     string    `json:"manga_id"`
	MangaSlug   string    `json:"manga_slug"`
	Number      string    `json:"number"`
	Title       string    `json:"title,omitempty"`
	Language    string    `json:"language"`
	PageCount   int       `json:"page_count"`
	PublishedAt time.Time `json:"published_at"`
}

// Page represents a single page of a chapter, including the integrity
// checksum the client should verify after downloading the bytes.
//
// Page는 chapter의 단일 page를 나타내며, 클라이언트가 다운로드 후 검증해야 하는
// 무결성 checksum 을 포함합니다.
type Page struct {
	Index    int    `json:"index"`
	URL      string `json:"url"`
	ByteSize int64  `json:"byte_size"`
	MimeType string `json:"mime_type"`
	Checksum string `json:"checksum"` // format: "sha-256:<base64>"
}

// Manifest is the page manifest the reader fetches before rendering pages.
//
// Manifest는 reader가 page를 렌더링하기 전에 가져오는 page manifest입니다.
type Manifest struct {
	ChapterID  string `json:"chapter_id"`
	MangaID    string `json:"manga_id"`
	MangaSlug  string `json:"manga_slug"`
	Number     string `json:"number"`
	ReadingDir string `json:"reading_dir"`
	PageCount  int    `json:"page_count"`
	Pages      []Page `json:"pages"`
}

// PageSource describes where a page's bytes live on disk and what type they
// are. It is used by services that need to read the raw image (e.g. image
// serving handler, checksum calculator).
//
// PageSource는 page의 raw 바이트가 디스크 어디에 있는지와 타입을 설명합니다.
// 이미지 서빙 핸들러, checksum 계산기 등이 사용합니다.
type PageSource struct {
	ChapterID string
	Index     int
	Path      string
	MimeType  string
	ByteSize  int64
}
