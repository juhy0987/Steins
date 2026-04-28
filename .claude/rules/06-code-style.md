# Code Style and Conventions

## Core Principles

### Readability First
- Code should be self-documenting
- Clear naming over clever code
- Simplicity over complexity
- Remove unnecessary code

### Minimal Comments
- Only comment WHY, not WHAT
- Code should explain itself through naming
- Remove commented-out code
- Avoid redundant comments

### No Over-Engineering
- Solve current problems, not future ones
- Avoid premature abstraction
- Delete unused code completely
- Keep it simple

## Go Style Guide

### Formatting

1. **Indentation**: tabs (gofmt default)
	- Run `gofmt -w` (or `goimports`) before committing.
	- Do NOT configure 2-space indentation — fights with `gofmt`.

2. **Line Length**: max 100 characters
	- Break long lines at logical points
	- Align function parameters vertically when needed

3. **Imports**: Group and order
	```go
	import (
		// Standard library
		"context"
		"fmt"
		"time"

		// External packages
		"github.com/go-chi/chi/v5"
		"github.com/rs/zerolog/log"

		// Internal packages
		"steins/internal/chapter"
		"steins/internal/storage/postgres"
	)
	```

4. **Blank Lines**: Strategic spacing
	```go
	func GetChapter(ctx context.Context, id string) (*Chapter, error) {
		if err := validateID(id); err != nil {
			return nil, err
		}

		ch, err := repo.FindByID(ctx, id)
		if err != nil {
			return nil, err
		}

		return ch, nil
	}
	```

### Naming Conventions

1. **Variables**: camelCase, descriptive
	```go
	// Good
	chapterCount := 10
	maxRetries := 3
	httpClient := &http.Client{}

	// Bad
	cc := 10
	max_retries := 3
	HTTPClient := &http.Client{}
	```

2. **Constants**: CamelCase (not SCREAMING_CASE)
	```go
	// Good
	const (
		DefaultTimeout = 30 * time.Second
		MaxPagesPerChapter = 500
	)

	// Bad
	const (
		DEFAULT_TIMEOUT = 30 * time.Second
		MAX_PAGES_PER_CHAPTER = 500
	)
	```

3. **Functions**: CamelCase
	- Exported: Start with capital letter
	- Unexported: Start with lowercase
	```go
	// Exported
	func GetChapter(id string) {}
	func ParsePageManifest(data []byte) {}

	// Unexported
	func validateID(id string) {}
	func decodeImage(r io.Reader) {}
	```

4. **Interfaces**: Short, descriptive names
	```go
	// Good
	type Reader interface {
		Read(ctx context.Context, id string) (*Chapter, error)
	}

	type Repository interface {
		Save(ctx context.Context, m *Manga) error
		Find(ctx context.Context, id string) (*Manga, error)
	}

	// Bad - don't add "Interface" suffix
	type ReaderInterface interface {}
	```

5. **Struct Names**: Singular nouns
	```go
	// Good
	type Manga struct {}
	type ChapterService struct {}

	// Bad
	type Mangas struct {}
	type ChapterServiceImpl struct {}
	```

### Error Handling

1. **Error Messages**: Lowercase, no punctuation
	```go
	// Good
	return fmt.Errorf("failed to fetch chapter: %w", err)

	// Bad
	return fmt.Errorf("Failed to fetch chapter.", err)
	```

2. **Early Returns**: Check errors immediately
	```go
	// Good
	func process() error {
		if err := validate(); err != nil {
			return err
		}

		if err := persist(); err != nil {
			return err
		}

		return nil
	}

	// Bad
	func process() error {
		err := validate()
		if err == nil {
			err = persist()
		}
		return err
	}
	```

3. **Named Return Values**: Only for documentation
	```go
	// Good - explicit returns
	func split(s string) (string, string) {
		parts := strings.Split(s, ":")
		return parts[0], parts[1]
	}

	// Bad - naked returns are confusing
	func split(s string) (head, tail string) {
		parts := strings.Split(s, ":")
		head = parts[0]
		tail = parts[1]
		return
	}
	```

### Function Design

1. **Small Functions**: Max 50 lines
	- One responsibility per function
	- Extract complex logic

2. **Parameter Count**: Max 5 parameters
	```go
	// Good
	type FetchOptions struct {
		Timeout time.Duration
		Limit   int
		Sort    string
	}

	func ListChapters(ctx context.Context, mangaID string, opts FetchOptions) {}

	// Bad
	func ListChapters(ctx context.Context, mangaID string, timeout time.Duration,
		limit int, sort string, language string, includeDrafts bool) {}
	```

3. **Context First**: Always first parameter
	```go
	// Good
	func GetChapter(ctx context.Context, id string) {}

	// Bad
	func GetChapter(id string, ctx context.Context) {}
	```

### Comments

**Language Policy**:
- **Primary**: Korean (한국어)
- **Secondary**: English (영어)
- **Encoding**: UTF-8
- **Style**: Mix Korean and English naturally for clarity

**Principles**:
- Keep comments minimal and essential
- Explain WHY, never WHAT
- Use Korean for main explanations, English for technical terms

1. **Function Comments**: Only for exported functions
	```go
	// GetChapter: 주어진 ID로 chapter를 조회합니다.
	// chapter가 존재하지 않으면 ErrNotFound를 반환합니다.
	func GetChapter(ctx context.Context, id string) (*Chapter, error) {
		// ...
	}

	// NewChapterService: 주어진 dependencies로 새로운 ChapterService를 생성합니다.
	func NewChapterService(repo Repository, storage ObjectStorage) *ChapterService {
		// ...
	}

	// unexported 함수는 복잡한 경우에만 주석 작성
	func validateChapterNumber(n float64) error {
		// ...
	}
	```

2. **Inline Comments**: Explain WHY, not WHAT
	```go
	// Good - 이유를 설명
	func retry() {
		// 서버 과부하 방지를 위해 재시도 전 대기
		time.Sleep(backoff)
	}

	// Good - 영어와 한글 혼용
	func processChapter() {
		// derivative가 이미 존재하면 idempotent 처리 (재시도 시 중복 작업 방지)
		if exists, _ := s.HasDerivatives(ctx, id); exists {
			return nil
		}
	}

	// Bad - 당연한 내용을 설명
	func retry() {
		// backoff 시간만큼 sleep
		time.Sleep(backoff)
	}
	```

3. **TODO Comments**: Action items with context
	```go
	// Good
	// TODO(username): rate limiter를 sliding-window로 교체 (burst 처리 정확도 개선)
	func applyRateLimit() {}

	// TODO: 페이지 prefetch hint를 manifest에 포함 (모바일 reader 성능)
	func buildManifest() {}

	// Bad - 맥락 없음
	// TODO: fix this
	func broken() {}
	```

4. **Package-Level Comments**: Korean + English
	```go
	// Package chapter는 만화 chapter 도메인의 service, repository,
	// 그리고 model 정의를 제공합니다.
	//
	// Package chapter provides the service, repository, and model
	// definitions for the manga chapter domain.
	//
	// 모든 chapter 관련 비즈니스 로직은 ChapterService를 통해 수행해야 하며,
	// 직접 Repository를 호출하는 것은 권장되지 않습니다.
	package chapter
	```

### Struct Design

1. **Field Ordering**: Group related fields
	```go
	type Manga struct {
		// Identity
		ID   string
		Slug string

		// Content
		Title       string
		Description string
		CoverKey    string

		// Classification
		Authors []string
		Genres  []string
		Status  Status

		// Metrics
		ViewCount int64
		Rating    float64

		// Timestamps
		PublishedAt time.Time
		UpdatedAt   time.Time
		CreatedAt   time.Time
	}
	```

2. **Struct Tags**: Aligned for readability
	```go
	type ChapterDTO struct {
		ID          string    `json:"id"           db:"id"`
		MangaID     string    `json:"manga_id"     db:"manga_id"`
		Number      float64   `json:"number"       db:"number"`
		Title       string    `json:"title"        db:"title"`
		PublishedAt time.Time `json:"published_at" db:"published_at"`
	}
	```

3. **Embedded Structs**: Use sparingly
	```go
	// Good - clear composition
	type ChapterService struct {
		chapters Repository
		storage  ObjectStorage
		queue    Enqueuer
	}

	// Avoid - unclear what's inherited
	type ChapterService struct {
		Repository
		ObjectStorage
	}
	```

### Package Design

1. **Package Names**: Short, lowercase, singular
	```go
	// Good
	package chapter
	package storage

	// Bad
	package chapters
	package chapter_storage
	```

2. **Package Organization**: By functionality
	```
	internal/
	└── chapter/
		├── service.go      # ChapterService
		├── repository.go   # Repository interface
		├── model.go        # Domain types
		└── errors.go       # Domain-specific errors
	```

3. **Internal Packages**: Hide implementation details
	- Use `internal/` for non-public packages
	- Expose only necessary types

### Concurrency

1. **Channel Direction**: Specify when possible
	```go
	func producer(out chan<- Page) {
		out <- page
	}

	func consumer(in <-chan Page) {
		page := <-in
	}
	```

2. **Context Handling**: Always check
	```go
	func process(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				work()
			}
		}
	}
	```

3. **WaitGroups**: Clear pattern
	```go
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			work()
		}()
	}

	wg.Wait()
	```

## Database Conventions

### SQL Style

1. **Keywords**: UPPERCASE
2. **Table/Column Names**: snake_case
3. **Indentation**: 2 spaces

```sql
-- Good
SELECT
  c.id,
  c.number,
  c.title,
  c.published_at
FROM chapters c
WHERE c.manga_id = $1
  AND c.status = 'ready'
ORDER BY c.number DESC
LIMIT 50;

-- Bad
select c.id, c.number, c.title from chapters c where c.manga_id = $1
and c.status = 'ready' order by c.number desc limit 50;
```

### Migration Files

```sql
-- migrations/0001_create_manga.up.sql
CREATE TABLE IF NOT EXISTS manga (
  id           TEXT PRIMARY KEY,
  slug         TEXT NOT NULL UNIQUE,
  title        TEXT NOT NULL,
  description  TEXT NOT NULL DEFAULT '',
  cover_key    TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT 'ongoing',
  language     CHAR(2) NOT NULL,
  reading_dir  TEXT NOT NULL DEFAULT 'rtl',
  view_count   BIGINT NOT NULL DEFAULT 0,
  rating       DOUBLE PRECISION NOT NULL DEFAULT 0,
  published_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_manga_status_updated ON manga(status, updated_at DESC);
CREATE INDEX idx_manga_language ON manga(language);
```

## Configuration Files

### YAML Style

1. **Indentation**: 2 spaces
2. **Keys**: snake_case
3. **Comments**: Explain non-obvious values

```yaml
# Good
server:
  http_addr: ":8080"
  read_timeout_ms: 5000
  write_timeout_ms: 10000

storage:
  provider: "s3"
  bucket: "steins-content"
  signed_url_ttl_s: 600  # 10 minutes

asynq:
  concurrency: 20
  queues:
    images: 6           # heavy CPU/memory work
    indexing: 3
    notifications: 2
    recommendations: 1
    cleanup: 1

# Bad
server:
    httpAddr: ":8080"
    ReadTimeoutMs: 5000
```

## Testing Conventions

### Test Function Names

```go
// Pattern: Test<Function>_<Scenario>_<Expected>
func TestGetChapter_ValidID_ReturnsChapter(t *testing.T) {}
func TestGetChapter_MissingID_ReturnsNotFound(t *testing.T) {}
func TestProcessImage_LargeFile_ReturnsTimeoutError(t *testing.T) {}
```

### Test Structure

```go
func TestChapterService_GetByID_Success(t *testing.T) {
	// Arrange
	repo := &mockChapterRepo{}
	repo.On("FindByID", mock.Anything, "ch-1").Return(&Chapter{ID: "ch-1"}, nil)
	svc := NewChapterService(repo, nil, nil)

	// Act
	got, err := svc.GetByID(context.Background(), "ch-1")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ch-1", got.ID)
}
```

### Table-Driven Tests

```go
func TestParseShelf(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Shelf
		wantErr bool
	}{
		{name: "reading", input: "reading", want: ShelfReading},
		{name: "completed", input: "completed", want: ShelfCompleted},
		{name: "invalid", input: "burning", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseShelf(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

## Anti-Patterns to Avoid

### 1. Magic Numbers
```go
// Bad
if pageCount > 500 {
	return err
}

// Good
const MaxPagesPerChapter = 500

if pageCount > MaxPagesPerChapter {
	return err
}
```

### 2. Deep Nesting
```go
// Bad
if a {
	if b {
		if c {
			if d {
				doSomething()
			}
		}
	}
}

// Good
if !a {
	return
}
if !b {
	return
}
if !c {
	return
}
if !d {
	return
}

doSomething()
```

### 3. Else After Return
```go
// Bad
if err != nil {
	return err
} else {
	process()
}

// Good
if err != nil {
	return err
}

process()
```

### 4. Unnecessary Variables
```go
// Bad
func isValid(s string) bool {
	result := len(s) > 0
	return result
}

// Good
func isValid(s string) bool {
	return len(s) > 0
}
```

### 5. Init Functions
```go
// Avoid - implicit initialization
func init() {
	db = connectDB()
}

// Prefer - explicit initialization
func New() *Service {
	return &Service{
		db: connectDB(),
	}
}
```

## Logging Standards

### Log Levels
```go
// DEBUG - development only
log.Debug().Str("chapter_id", id).Msg("assembling page manifest")

// INFO - normal operations
log.Info().Str("chapter_id", id).Int("pages", n).Msg("chapter processing completed")

// WARN - unexpected but handled
log.Warn().Err(err).Msg("retry attempt")

// ERROR - operation failed
log.Error().Err(err).Str("chapter_id", id).Msg("failed to process chapter")
```

### Structured Logging
```go
// Good - structured fields
log.Info().
	Str("route", "/api/v1/chapters/{id}/pages").
	Str("chapter_id", chapterID).
	Int("status", http.StatusOK).
	Dur("duration", elapsed).
	Msg("chapter pages served")

// Bad - string interpolation
log.Info().Msgf("Served chapter %s in %v", chapterID, elapsed)
```

### Context in Logs
```go
// Include relevant context
log.Error().
	Err(err).
	Str("chapter_id", ch.ID).
	Int("version", ch.Version).
	Str("error_code", "IMG_001").
	Msg("page decode failed")

// Not just the error
log.Error().Err(err).Msg("error")
```

## Git Commit Conventions

### Commit Message Format
```
[{카테고리}]: {변경 내용}
```

### 카테고리 (Categories)

- **FEAT**: feature, 기능 구현 및 추가
- **FIX**: fix, 버그 수정
- **REFAC**: refactor, 구조 변경, 메소드 구조 변경 및 리팩토링
- **DOCS**: documentation, 문서 작업 및 프롬프트 변경, 주석 등 설명 요소 작성
- **CHORE**: chore, 빌드·CI·도구·의존성 등 잡무 (코드 외 부가 작업)

### 변경 내용 작성 규칙

**⚠️ 중요: 모든 커밋 메시지는 한국어로 작성해야 합니다.**

1. **언어**: 반드시 한국어로 작성 (영어 사용 금지)
2. **형식**: 명사형 종결 (예: "구현", "수정", "추가")
3. **내용**: 변경 내용의 전체적인 요약, 각 모듈 단위의 변경점을 명확히 기술

### Examples

**기능 구현 및 추가:**
```
[FEAT]: chapter page manifest API 구현

- ChapterService.GetPageManifest 메소드 추가
- /api/v1/chapters/{id}/pages 핸들러 구현
- Redis 기반 page manifest 캐시 (TTL 10m)
```

```
[FEAT]: Asynq 기반 image processing worker 구현

- process_chapter 작업 핸들러 추가
- WebP variant (thumbnail/preview/web) 생성 로직
- idempotent 재처리 지원
```

**버그 수정:**
```
[FIX]: 페이지 reading progress index clamp 수정

- chapter republish 시 page_index가 새 page_count를 초과하던 문제 해결
- UpdateProgress에서 boundary check 추가
```

**구조 변경 및 리팩토링:**
```
[REFAC]: ChapterService 트랜잭션 경계 정리

- 핸들러에서 직접 호출되던 repo 메소드를 service로 이동
- 트랜잭션을 service 레이어에서 일괄 관리
```

**문서 작업:**
```
[DOCS]: Reader API 문서 및 사용 예제 작성

- /api/v1/chapters 엔드포인트 GoDoc 주석 추가
- examples/reader_usage.go 작성
- README.md에 Quick Start 섹션 추가
```

### Branch Naming Convention

```
{카테고리}/#{이슈번호}/{핵심-변경-대상-요약}
```

#### 브랜치 대분류

- **feature/**: 새로운 기능 구현 및 추가 (FEAT 커밋 대응)
- **fix/**: 버그 수정 (FIX 커밋 대응)
- **refactor/**: 구조 변경 및 리팩토링 (REFAC 커밋 대응)
- **docs/**: 문서 작업 (DOCS 커밋 대응)
- **chore/**: 빌드·CI·도구 등 잡무 (CHORE 커밋 대응)

#### 브랜치명 작성 규칙

1. **형식**: `{카테고리}/#{이슈번호}/{핵심-변경-대상-요약}`
2. **이슈번호**: GitHub 이슈 번호를 `#` 접두사와 함께 표기 (예: `#15`)
3. **핵심 변경 대상 요약**: 영문 소문자, 단어 구분은 하이픈(-), 30자 이내
4. **내용**: 변경 대상 파일 또는 모듈명 중심으로 간결하게 표현

#### Branch Name Examples

```bash
feature/#15/chapter-page-manifest
feature/#16/asynq-image-worker
feature/#17/meilisearch-indexer
feature/#18/reading-progress-api
feature/#19/redis-rate-limiter
feature/#20/jwt-auth-middleware
feature/#21/library-shelves

fix/#7/progress-index-clamp
fix/#9/duplicate-chapter-detection

refactor/#4/chapter-service-tx-boundary
refactor/#8/extract-image-processor

docs/#2/reader-api-documentation
docs/#9/git-conventions
```

## File Organization

### File Naming
- Lowercase, underscore-separated: `chapter_repository.go`
- Test files: `chapter_repository_test.go`
- Implementation-specific: `image_jpeg.go`, `image_webp.go`

### File Structure
```go
// 1. Package declaration
package chapter

// 2. Imports
import (
	"context"
	"fmt"
)

// 3. Constants
const (
	DefaultTimeout = 30 * time.Second
)

// 4. Types
type Service struct {
	repo Repository
}

// 5. Constructor
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// 6. Public methods
func (s *Service) GetByID(ctx context.Context, id string) (*Chapter, error) {
	return s.repo.FindByID(ctx, id)
}

// 7. Private methods
func (s *Service) buildManifest(ch *Chapter) *Manifest {
	return nil
}

// 8. Helper functions
func validateNumber(n float64) error {
	return nil
}
```

## Performance Guidelines

### 1. Use String Builders
```go
// Good
var sb strings.Builder
for _, s := range items {
	sb.WriteString(s)
}
result := sb.String()

// Bad
var result string
for _, s := range items {
	result += s
}
```

### 2. Pre-allocate Slices
```go
// Good
pages := make([]*Page, 0, expectedSize)

// Bad
var pages []*Page
```

### 3. Use sync.Pool for Frequent Allocations
```go
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func encodePage(p *Page) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	// Use buffer
	return buf.Bytes(), nil
}
```

## Documentation

### Language Requirements

**Code Documentation** (GoDoc):
- **Primary**: English (영어)
- **Secondary**: Korean (한국어)
- **Format**: English first, then Korean explanation
- **Encoding**: UTF-8

**Documentation Files**:
- **MUST** write all documentation in English first
- **MUST** provide Korean translation in separate directory
- **Directory structure**:
  ```
  docs/
  ├── en/           # English documentation (primary)
  │   ├── README.md
  │   ├── architecture.md
  │   └── api.md
  └── ko/           # Korean translation
      ├── README.md
      ├── architecture.md
      └── api.md
  ```
- **Naming convention**:
  - English: `docs/en/<filename>.md`
  - Korean: `docs/ko/<filename>.md`
- **Content**: Keep both versions synchronized

**README Files** (Root level):
- `README.md` - **MUST** be in English
- Link to Korean version: `docs/ko/README.md`
- Include language selector at the top:
  ```markdown
  # Steins

  **[한국어](docs/ko/README.md)** | English
  ```

### Package Documentation

```go
// Package chapter provides the chapter domain — service, repository,
// and model definitions for managing manga chapters.
//
// chapter 패키지는 만화 chapter 도메인의 service, repository,
// 그리고 model 정의를 제공합니다.
//
// All chapter-related business logic must go through ChapterService.
// Direct Repository calls from handlers are not recommended.
//
// 모든 chapter 관련 비즈니스 로직은 ChapterService를 통해 수행해야 하며,
// 핸들러에서 직접 Repository를 호출하는 것은 권장되지 않습니다.
package chapter
```

### Type Documentation

```go
// Manga represents a manga series with metadata, classification,
// and aggregated metrics.
//
// Manga는 만화 시리즈를 나타내며 메타데이터, 분류, 집계 metrics를 포함합니다.
type Manga struct {
	ID    string
	Title string
}

// Chapter represents a single chapter within a manga series.
//
// Chapter는 만화 시리즈의 단일 chapter를 나타냅니다.
type Chapter struct {
	ID      string
	MangaID string
	Number  float64
}
```

### Function Documentation

```go
// GetPageManifest returns the page manifest for a chapter.
// It returns ErrNotFound if the chapter does not exist or is not in `ready` status.
//
// GetPageManifest는 chapter의 page manifest를 반환합니다.
// chapter가 존재하지 않거나 `ready` 상태가 아니면 ErrNotFound를 반환합니다.
//
// Example:
//   manifest, err := svc.GetPageManifest(ctx, "ch-1")
//   if errors.Is(err, ErrNotFound) {
//     // Handle not found
//   }
func (s *ChapterService) GetPageManifest(ctx context.Context, id string) (*Manifest, error) {
	// ...
}
```

### Example Code

```go
// Example usage of the chapter service.
//
// ChapterService 사용 예시.
func ExampleChapterService_GetPageManifest() {
	svc := NewChapterService(repo, storage, queue)

	manifest, err := svc.GetPageManifest(context.Background(), "ch-1")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(len(manifest.Pages))
	// Output: 24
}
```

### README Structure

**README.md** (Root - English):
```markdown
# Steins

**[한국어](docs/ko/README.md)** | English

> Web-based manga reading platform

## Overview

Steins is a Go-based manga reading service that provides a fast,
reader-first experience: catalog browsing, chapter reading,
user libraries, and search.

## Key Features

- Catalog and chapter browsing
- Reader with paged / vertical-scroll / RTL modes
- User libraries, bookmarks, and reading progress
- Background image processing pipeline
- Full-text search

## Getting Started

### Requirements

- Go 1.22+
- PostgreSQL 15+
- Redis 7+
- S3-compatible object storage

### Installation

\`\`\`bash
git clone https://github.com/example/steins
cd steins
make install
\`\`\`

## Documentation

**English**:
- [Architecture](docs/en/architecture.md)
- [API Implementation](docs/en/api-implementation.md)
- [Code Style](docs/en/code-style.md)

**한국어**:
- [아키텍처](docs/ko/architecture.md)
- [API 구현](docs/ko/api-implementation.md)
- [코드 스타일](docs/ko/code-style.md)

## License

MIT
```

**docs/ko/README.md** (Korean Translation):
```markdown
# Steins

한국어 | **[English](../../README.md)**

> 웹 기반 만화 읽기 플랫폼

## 개요

Steins는 카탈로그 탐색, chapter 읽기, 사용자 라이브러리, 검색을 제공하는
reader-first 만화 서비스입니다.

## 주요 기능

- 카탈로그 및 chapter 탐색
- 페이지 / 세로 스크롤 / RTL 모드 reader
- 사용자 라이브러리, 북마크, 읽기 진행 상황
- 백그라운드 이미지 처리 파이프라인
- 전문 검색

## 시작하기

### 요구사항

- Go 1.22+
- PostgreSQL 15+
- Redis 7+
- S3 호환 object storage

### 설치

\`\`\`bash
git clone https://github.com/example/steins
cd steins
make install
\`\`\`

## 문서

**English**:
- [Architecture](../en/architecture.md)
- [API Implementation](../en/api-implementation.md)
- [Code Style](../en/code-style.md)

**한국어**:
- [아키텍처](architecture.md)
- [API 구현](api-implementation.md)
- [코드 스타일](code-style.md)

## 라이선스

MIT
```

### Documentation Best Practices

1. **Write English First**
   - All documentation **MUST** be written in English first
   - English version is the source of truth
   - Store in `docs/en/` directory

2. **Translate to Korean**
   - Create Korean translation after English is complete
   - Store in `docs/ko/` directory
   - Maintain same file structure as English
   - Keep synchronized with English updates

3. **Directory Organization**
   ```
   project/
   ├── README.md              # English (root level)
   ├── docs/
   │   ├── en/                # English documentation (source)
   │   │   ├── README.md
   │   │   ├── architecture.md
   │   │   ├── api.md
   │   │   ├── deployment.md
   │   │   └── troubleshooting.md
   │   └── ko/                # Korean translation
   │       ├── README.md
   │       ├── architecture.md
   │       ├── api.md
   │       ├── deployment.md
   │       └── troubleshooting.md
   └── .claude/
       └── rules/             # Development rules (English only)
   ```

4. **Code Comments**
   - Use English for GoDoc (exported functions/types)
   - Add Korean explanation below English
   - Mix languages naturally in inline comments

5. **API Documentation**
   - Generate with `godoc` (English)
   - Provide Korean translation in `docs/ko/api.md`

6. **Architecture Docs**
   - Write in English first (`docs/en/architecture.md`)
   - Translate to Korean (`docs/ko/architecture.md`)

7. **Changelogs**
   - Write in English (version control standard)
   - Provide Korean translation in separate section

8. **Translation Workflow**
   - Step 1: Write complete English documentation
   - Step 2: Review and approve English version
   - Step 3: Translate to Korean, maintaining structure
   - Step 4: Review Korean translation for accuracy
   - Step 5: Link both versions with language selector

9. **Language Selector**
   - Include at the top of every document
   - English: `**[한국어](../ko/same-file.md)** | English`
   - Korean: `한국어 | **[English](../en/same-file.md)**`

10. **Update Policy**
    - When updating documentation:
      1. Update English version first
      2. Update Korean translation to match
      3. Mark in commit message: `[DOCS]: [기능명] 문서 업데이트 (en+ko)`

## Code Review Checklist

Before submitting code for review, ensure:

- [ ] No commented-out code
- [ ] No unnecessary variables
- [ ] No magic numbers
- [ ] Error handling is complete (`%w` wrapping where appropriate)
- [ ] Functions are small (< 50 lines)
- [ ] No deep nesting (< 4 levels)
- [ ] Tests are included
- [ ] Logging includes context
- [ ] Code is self-documenting
- [ ] Comments explain WHY, not WHAT
