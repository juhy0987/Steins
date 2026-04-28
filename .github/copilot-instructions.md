# GitHub Copilot Instructions

## Language
- All responses and code review comments MUST be written in **Korean (한국어)**
- Technical terms (Go, Redis, gRPC, Asynq 등) may remain in English
- Code itself (identifiers, GoDoc comments) follows the conventions below

---

## Project Overview

**Steins** is a Go-based manga reading platform — a reader-first web service for browsing series, reading chapters, and managing personal libraries.

```
┌─────────────────────────────────────────┐
│     Edge / CDN (image delivery)         │
├─────────────────────────────────────────┤
│     HTTP API Layer (REST)               │
│  (Handlers, middleware, validation)     │
├─────────────────────────────────────────┤
│     Application Service Layer           │
│  (Manga, Chapter, User, Library)        │
├─────────────────────────────────────────┤
│     Background Worker Layer             │
│  (Image processing, indexing, jobs)     │
├─────────────────────────────────────────┤
│     Domain / Repository Layer           │
├─────────────────────────────────────────┤
│     Storage Layer                       │
│  (PostgreSQL, Redis, Object Storage)    │
└─────────────────────────────────────────┘
```

**Tech Stack**: Go 1.22+, PostgreSQL 15+, Redis 7+, S3-compatible object storage, Asynq, Meilisearch (or PostgreSQL FTS as starter)

---

## Branch Naming Convention

```
{category}/#{issue-number}/{short-kebab-summary}
```

| Category | 용도 |
|----------|------|
| `feature/` | 새로운 기능 구현 및 추가 |
| `fix/` | 버그 수정 |
| `refactor/` | 구조 변경 및 리팩토링 |
| `docs/` | 문서 작업 |
| `chore/` | 빌드·CI·도구 등 잡무 |

**규칙:**
- 이슈 번호는 `#` 접두사 포함 (예: `#15`)
- 요약은 영문 소문자 + 하이픈(-) + 30자 이내
- 변경 대상 파일 또는 모듈명 중심으로 간결하게 표현

**예시:**
```
feature/#15/chapter-page-manifest
feature/#16/asynq-image-worker
feature/#17/meilisearch-indexer
feature/#18/reading-progress-api
feature/#19/redis-rate-limiter
fix/#7/progress-index-clamp
fix/#9/duplicate-chapter-detection
refactor/#4/chapter-service-tx-boundary
docs/#2/reader-api-documentation
chore/#3/golangci-lint-version-bump
```

---

## Git Commit Convention

### Format

```
[{CATEGORY}]: {변경 내용}
```

### Categories

| Category | 용도 |
|----------|------|
| `FEAT` | 기능 구현 및 추가 |
| `FIX` | 버그 수정 |
| `REFAC` | 구조 변경, 리팩토링 |
| `DOCS` | 문서 작업, 주석, 프롬프트 변경 |
| `CHORE` | 빌드·CI·도구·의존성 등 잡무 (코드 외 부가 작업) |

### 작성 규칙

> **⚠️ 커밋 메시지는 반드시 한국어로 작성**

1. 언어: 한국어 (영어 사용 금지)
2. 형식: 명사형 종결 (예: "구현", "수정", "추가")
3. 내용: 변경 사항의 전체 요약 + 모듈별 변경점 명시

### 예시

```
[FEAT]: chapter page manifest API 구현

- ChapterService.GetPageManifest 메소드 추가
- /api/v1/chapters/{id}/pages 핸들러 구현
- Redis 기반 page manifest 캐시 (TTL 10m)
```

```
[FIX]: 페이지 reading progress index clamp 수정

- chapter republish 시 page_index가 새 page_count를 초과하던 문제 해결
- UpdateProgress에서 boundary check 추가
```

```
[REFAC]: ChapterService 트랜잭션 경계 정리

- 핸들러에서 직접 호출되던 repo 메소드를 service로 이동
- 트랜잭션을 service 레이어에서 일괄 관리
```

```
[CHORE]: golangci-lint 버전 v1.64.8 로 업데이트

- ci-quality.yml lint job 의 binary 버전 고정
- 로컬 Makefile 의 lint 타겟에 동일 버전 명시
```

```
[DOCS]: Reader API 문서 및 사용 예제 작성

- GoDoc 주석 추가
- README.md Quick Start 섹션 추가
```

---

## Code Style Guide (Go)

### Formatting

- **Indentation**: tabs (gofmt default)
- **Line Length**: 최대 100자
- **Imports**: 표준 라이브러리 → 외부 패키지 → 내부 패키지 순으로 그룹 분리

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

### Naming Conventions

| 대상 | 규칙 | 예시 |
|------|------|------|
| 변수 | camelCase | `chapterCount`, `httpClient` |
| 상수 | CamelCase (SCREAMING_CASE 금지) | `DefaultTimeout`, `MaxPagesPerChapter` |
| Exported 함수 | 대문자 시작 CamelCase | `GetChapter`, `ParsePageManifest` |
| Unexported 함수 | 소문자 시작 camelCase | `validateID`, `decodeImage` |
| 인터페이스 | 짧고 명확하게 (Interface 접미사 금지) | `Reader`, `Repository` |
| 구조체 | 단수 명사 | `Manga`, `Chapter` |
| 패키지 | 소문자, 단수 | `chapter`, `storage` |

### Function Design

- 함수당 최대 50줄
- 파라미터 최대 5개 (초과 시 Options struct로 묶기)
- `context.Context`는 항상 첫 번째 파라미터
- Early Return 패턴 사용

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
```

### Anti-Patterns (금지)

```go
// ❌ Magic numbers
if pageCount > 500 { ... }
// ✅ Named constants
const MaxPagesPerChapter = 500
if pageCount > MaxPagesPerChapter { ... }
```

```go
// ❌ Deep nesting (4단계 초과)
if a { if b { if c { if d { doSomething() } } } }
// ✅ Early return
if !a { return }
if !b { return }
doSomething()
```

```go
// ❌ else after return
if err != nil { return err } else { process() }
// ✅
if err != nil { return err }
process()
```

```go
// ❌ init() 함수 사용 (암묵적 초기화)
func init() { db = connectDB() }
// ✅ 명시적 초기화
func New() *Service { return &Service{db: connectDB()} }
```

```go
// ❌ Unnecessary variable
func isValid(s string) bool {
	result := len(s) > 0
	return result
}
// ✅
func isValid(s string) bool {
	return len(s) > 0
}
```

### Struct Design

관련 필드를 그룹핑, struct tags는 정렬:

```go
type Manga struct {
	// Identity
	ID   string
	Slug string

	// Content
	Title       string
	Description string

	// Classification
	Genres []string
	Status Status

	// Timestamps
	PublishedAt time.Time
	CreatedAt   time.Time
}

// Struct tags — 정렬 필수
type ChapterDTO struct {
	ID          string    `json:"id"           db:"id"`
	MangaID     string    `json:"manga_id"     db:"manga_id"`
	PublishedAt time.Time `json:"published_at" db:"published_at"`
}
```

### Concurrency

```go
// Channel direction 명시
func producer(out chan<- Page) { out <- page }
func consumer(in <-chan Page)  { p := <-in }

// Context 취소 항상 확인
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

---

## Comments Convention

**언어 정책:**
- 인라인 주석: **한국어 우선**, 영어 기술 용어 혼용 허용
- GoDoc (exported 심볼): **영어 우선**, 한국어 설명 아래에 추가
- 주석 원칙: **WHY만 설명** (WHAT은 코드로 표현, 당연한 내용 금지)

```go
// GetPageManifest returns the page manifest for a chapter.
// GetPageManifest는 chapter의 page manifest를 반환합니다.
func GetPageManifest(ctx context.Context, id string) (*Manifest, error) {
	// derivative가 이미 존재하면 idempotent 처리 (재시도 시 중복 작업 방지)
	if exists, _ := storage.HasDerivatives(ctx, id); exists {
		return cachedManifest(id), nil
	}
	// ...
}
```

**TODO 형식:**
```go
// TODO(username): rate limiter를 sliding-window로 교체 (burst 처리 정확도 개선)
// TODO: 페이지 prefetch hint를 manifest에 포함 (모바일 reader 성능)
```

---

## Error Handling

- Production 코드에서 `panic` 사용 금지 (`recover()`는 top-level handler에서만)
- 에러는 `fmt.Errorf("context: %w", err)` 로 래핑
- 에러 메시지: 소문자 시작, 마침표 없음

```go
// Good
return fmt.Errorf("failed to fetch chapter: %w", err)
// Bad
return fmt.Errorf("Failed to fetch chapter.", err)
```

### Error Categories & Codes

```
VAL_001~007  : 입력 검증 오류 (필수 필드, 형식, 파일 크기, MIME)
AUTH_001~005 : 인증/인가 오류
*_NOT_FOUND  : 리소스 없음 (CHAPTER_NOT_FOUND, MANGA_NOT_FOUND, ...)
NET_001~003  : 네트워크 오류 (Connection refused, timeout, DNS)
HTTP_4xx/5xx : HTTP 상태 오류
STORAGE_001~004 : object storage 오류
DB_001~004   : 데이터베이스 오류
QUEUE_001~002, WORKER_001~002 : Asynq 작업 오류
IMG_001~004  : 이미지 처리 오류
SEARCH_001~002 : 검색 백엔드 오류
```

### Retry 원칙

| 오류 유형 | 최대 재시도 | 초기 대기 | 비고 |
|-----------|-------------|-----------|------|
| Network/Timeout | 3회 | 1초 | exponential backoff + jitter |
| RateLimit | 5회 | 10초 | exponential backoff |
| Permanent (404, 403, validation) | 0회 | — | 즉시 반환 |
| Worker idempotent task | 5회 | 1분 | Asynq 기본 정책 |

---

## Logging (zerolog)

**Log Levels:**

| Level | 용도 |
|-------|------|
| `DEBUG` | 개발/진단용 상세 정보 (요청/응답, manifest 조립 단계) |
| `INFO` | 정상 운영 (요청 완료, 작업 완료, 시작/종료) |
| `WARN` | 처리는 됐지만 비정상 (재시도, fallback) |
| `ERROR` | 작업 실패 |
| `FATAL` | 복구 불가능한 오류 (설정 오류, DB 치명 오류) |

**Structured logging 사용 (string interpolation 금지):**

```go
// Good
log.Info().
	Str("route", "/api/v1/chapters/{id}/pages").
	Str("chapter_id", chapterID).
	Int("status", http.StatusOK).
	Dur("duration", elapsed).
	Msg("chapter pages served")

// Bad
log.Info().Msgf("Served chapter %s in %v", chapterID, elapsed)
```

**항상 관련 컨텍스트 포함:**
```go
log.Error().
	Err(err).
	Str("chapter_id", ch.ID).
	Int("version", ch.Version).
	Str("error_code", "IMG_001").
	Msg("page decode failed")
```

---

## Testing

### Naming Pattern

```go
// Test{Function}_{Scenario}_{Expected}
func TestGetChapter_ValidID_ReturnsChapter(t *testing.T) {}
func TestGetChapter_MissingID_ReturnsNotFound(t *testing.T) {}
func TestProcessImage_LargeFile_ReturnsTimeoutError(t *testing.T) {}
```

### Test Structure (AAA)

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

### Coverage 기준

| 대상 | 최소 커버리지 |
|------|---------------|
| 핵심 패키지 | 70% |
| 핸들러/서비스/이미지 처리 | 90% |
| 에러 핸들링 경로 | 100% |

### Test 파일 위치

소스 파일과 **같은 디렉토리에 두지 않음** — 별도 `test/` 디렉토리 사용:

```
test/
├── internal/
│   ├── api_handler/
│   │   ├── chapter_test.go
│   │   └── reader_test.go
│   ├── chapter/
│   │   └── service_test.go
│   └── storage_postgres/
│       └── chapter_repo_test.go
└── pkg/
	├── config/
	│   └── config_test.go
	└── httpx/
		└── error_test.go
```

패키지 선언: `package <name>_test` (외부 테스트 패키지)

---

## Database Conventions

### SQL Style

- Keywords: UPPERCASE
- Table/Column Names: snake_case
- Indentation: 2 spaces

```sql
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
```

---

## Configuration Files (YAML)

- Indentation: 2 spaces
- Keys: snake_case
- 비자명한 값에는 단위/의미 주석 추가

```yaml
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
```

---

## File Organization

### File Naming

- 소문자, 언더스코어 구분: `chapter_repository.go`
- 테스트 파일: `chapter_repository_test.go`
- 구현 종류별: `image_jpeg.go`, `image_webp.go`

### File Structure (내부 순서)

```go
// 1. Package declaration
// 2. Imports (grouped)
// 3. Constants
// 4. Types (interfaces → structs)
// 5. Constructors (New...)
// 6. Public methods
// 7. Private methods
// 8. Helper functions
```

### Package Organization

```
internal/
└── chapter/
	├── service.go      # ChapterService
	├── repository.go   # Repository interface
	├── model.go        # Domain types
	└── errors.go       # Domain-specific errors
```

---

## Documentation

- `README.md` (root): **영어**로 작성, 상단에 한국어 링크 포함
- 한국어 번역: `docs/ko/README.md`
- GoDoc: 영어 우선, 한국어 설명 병기
- 문서 업데이트 시 영어 먼저, 한국어 번역 후 동기화

문서 관련 커밋 메시지:
```
[DOCS]: [기능명] 문서 업데이트 (en+ko)
```

---

## Issue Convention

### 이슈 타이틀 형식

```
[{CATEGORY}] 이슈 타이틀
```

| Category | 용도 | 라벨 | 템플릿 |
|----------|------|------|--------|
| `FEATURE` | 새로운 기능 요청 및 구현 | `feature` | `feature.md` |
| `BUG` | 버그 리포트 및 수정 | `bug` | `bug.md` |
| `REFACTOR` | 코드 구조 개선, 리팩토링 | `refactor` | `refactor.md` |
| `CHORE` | 문서 작업, 의존성 업데이트 등 코드 외 부가 작업 | `chore` | `chore.md` |

**예시:**
```
[FEATURE] Chapter page manifest API 구현
[BUG] reading progress index clamp 누락
[REFACTOR] ChapterService 트랜잭션 경계 정리
[CHORE] go.mod 의존성 업데이트
[CHORE] Reader API 문서 작성
```

### 이슈 템플릿별 작성 규칙

**FEATURE** (`.github/ISSUE_TEMPLATE/feature.md`):

| 섹션 | 설명 |
|------|------|
| 어떤 기능인가요? | 구현할 기능의 목적과 배경을 간략히 서술 |
| 무엇을 하나요? | 구현 단위를 task 체크리스트로 나열 |
| 참고 자료 | 관련 문서, 링크, 설계 자료 |

**BUG** (`.github/ISSUE_TEMPLATE/bug.md`):

| 섹션 | 설명 |
|------|------|
| 무엇을 수정하나요? | 버그 현상과 예상 동작을 서술 |
| 참고 자료 | 로그, 스크린샷, 재현 방법 |

**REFACTOR** (`.github/ISSUE_TEMPLATE/refactor.md`):

| 섹션 | 설명 |
|------|------|
| 무엇을 개선하나요? | 현재 문제점과 개선 목표를 서술 |
| 무엇을 하나요? | 개선 작업을 task 체크리스트로 나열 |
| 참고 자료 | 관련 문서, 링크 |

**CHORE** (`.github/ISSUE_TEMPLATE/chore.md`):

| 섹션 | 설명 |
|------|------|
| 어떤 작업인가요? | 작업의 목적과 배경을 간략히 서술 |
| 무엇을 하나요? | 작업 항목을 task 체크리스트로 나열 |
| 참고 자료 | 관련 문서, 링크 |

### 이슈 ↔ 브랜치 ↔ PR ↔ 커밋 연결

```
이슈: [FEATURE] Chapter page manifest API 구현 (이슈 #15)
  ↓
브랜치: feature/#15/chapter-page-manifest
  ↓
커밋: [FEAT]: ChapterService GetPageManifest 메소드 추가
  ↓
PR 타이틀: [FEAT#15] Chapter page manifest API 구현
PR 라벨: feature
```

> 이슈 카테고리(FEATURE)와 커밋/브랜치/PR 카테고리(FEAT)는 약어가 다를 수 있다. 위 매핑을 기준으로 맞춘다.

| 이슈 | 브랜치 prefix | 커밋/PR |
|------|---------------|---------|
| `FEATURE` | `feature/` | `FEAT` |
| `BUG` | `fix/` | `FIX` |
| `REFACTOR` | `refactor/` | `REFAC` |
| `CHORE` | `chore/` 또는 `docs/` | `CHORE` (코드 외 잡무) 또는 `DOCS` (순수 문서 작업) |

---

## Pull Request Convention

### PR 타이틀 형식

CI (`PR Title Lint`) 가 정규식으로 엄격 강제합니다:
`^\[(FEAT|FIX|REFAC|DOCS|CHORE)#[0-9]+\]:? .+`

```
[{CATEGORY}#{이슈번호}] PR 타이틀
[{CATEGORY}#{이슈번호}]: PR 타이틀   (콜론 형태도 허용)
```

| Category | 용도 |
|----------|------|
| `FEAT` | 기능 구현 및 추가 |
| `FIX` | 버그 수정 |
| `REFAC` | 구조 변경, 리팩토링 |
| `DOCS` | 문서 작업 |
| `CHORE` | 빌드·CI·도구 등 잡무 |

**통과 예시:**
```
[FEAT#15] Chapter page manifest API 구현
[FEAT#15]: Chapter page manifest API 구현
[FIX#7] reading progress index clamp 수정
[REFAC#4] ChapterService 트랜잭션 경계 정리
[DOCS#2] Reader API 문서 작성
[CHORE#117] CI golangci-lint 버전 업데이트
```

**거부 예시 (CI 실패):**
- `[FEAT]: 이슈번호 누락` — `#이슈번호` 필수
- `[FEAT 119]: # 대신 공백` — 반드시 `#` 사용
- `[FEAT#abc]: 숫자 아닌 이슈번호` — 숫자만 허용
- `[FEATXX#1]: 잘못된 카테고리` — 위 5개만 허용
- `[FEAT#1]설명` — `]` 또는 `:` 뒤 공백 필수
- `feat#1: 소문자` — 카테고리는 대문자만 허용

### PR 본문 — 템플릿 작성 규칙

PR 본문은 `.github/PULL_REQUEST_TEMPLATE.md` 폼을 그대로 사용한다.

```markdown
## 연관 이슈
- #{이슈번호}

## 구현 내용
- {변경사항 1}
- {변경사항 2}
- ...

## TODO
-

## 논의 사항
-
```

**섹션별 작성 원칙:**

| 섹션 | 규칙 |
|------|------|
| **연관 이슈** | 반드시 연관 이슈 번호 링크 (`- #15`) |
| **구현 내용** | 변경사항과 관련 핵심 함수/구조체를 명시. **반드시 작성** |
| **TODO** | 보완이 필요한 항목이 없으면 그대로 둠 (`- `) |
| **논의 사항** | 논의가 필요한 항목이 없으면 그대로 둠 (`- `) |

> **⚠️ 작업 내용(구현 내용)만 채우고, TODO·논의 사항에 기재할 내용이 없으면 빈 항목(`- `)을 지우지 않고 그대로 남긴다.**

### 라벨

PR 카테고리에 맞는 라벨을 지정:

| 라벨 | 적용 조건 |
|------|-----------|
| `feature` | FEAT 카테고리 PR |
| `bug` | FIX 카테고리 PR |
| `refactor` | REFAC 카테고리 PR |
| `documentation` | DOCS 카테고리 PR |
| `breaking change` | 하위 호환성을 깨는 변경 포함 시 추가 |
| `wip` | 작업이 완료되지 않은 draft PR |

### Tasks (체크리스트)

PR 생성 시 구현 내용에 따라 관련 task를 자동으로 추가:

**FEAT PR:**
- [ ] 인터페이스/구조체 정의 완료
- [ ] 핵심 로직 구현 완료
- [ ] 단위 테스트 작성 (커버리지 기준 충족)
- [ ] GoDoc 주석 작성 (exported 심볼)
- [ ] 통합 테스트 작성 (해당하는 경우)

**FIX PR:**
- [ ] 버그 재현 테스트 작성
- [ ] 수정 사항 구현
- [ ] 기존 테스트 통과 확인
- [ ] 회귀 테스트 추가

**REFAC PR:**
- [ ] 기존 동작 변경 없음 확인
- [ ] 기존 테스트 통과 확인
- [ ] 불필요한 코드 제거

**DOCS PR:**
- [ ] 영어 문서 작성/수정
- [ ] 한국어 번역 동기화

### PR 예시

```markdown
## 연관 이슈
- #15

## 구현 내용
- `internal/chapter/` 패키지에 ChapterService 구현
- `internal/api/handler/chapter/` 에 page manifest 핸들러 추가
- Redis 기반 manifest 캐시 (TTL 10m) 적용
- 핸들러 단위 테스트, 서비스 단위 테스트 작성

## TODO
-

## 논의 사항
-
```

---

## Code Review Checklist

PR 제출 전 확인:

- [ ] 커밋 메시지가 한국어 + `[CATEGORY]:` 형식을 따름
- [ ] 브랜치명이 `{category}/#{issue}/{kebab-summary}` 형식을 따름
- [ ] 주석 처리된 코드 없음
- [ ] 불필요한 변수 없음
- [ ] Magic number 없음 (상수 사용)
- [ ] 에러 핸들링 완결 (`%w` 래핑 포함)
- [ ] 함수 크기 50줄 이하
- [ ] 중첩 깊이 4단계 이하
- [ ] `context.Context` 첫 번째 파라미터
- [ ] 테스트 포함 (커버리지 기준 충족)
- [ ] 로그에 관련 컨텍스트 포함
- [ ] `init()` 함수 미사용
- [ ] 코드가 자기 설명적 (WHAT 주석 금지)
