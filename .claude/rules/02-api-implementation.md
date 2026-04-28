# API and Reader Service Implementation Rules

## Layering

Steins uses a strict three-layer separation inside each domain package:

```
handler  →  service  →  repository
              ↓
        external systems
        (object storage, search, queue)
```

- **Handler**: HTTP I/O only — decode request, validate input, call service, encode response.
- **Service**: business logic, transactions, orchestration of repositories and external systems.
- **Repository**: data access — SQL, Redis commands, object storage calls.

Handlers MUST NOT call repositories directly. Services MUST NOT depend on `net/http`.

## HTTP API Conventions

### URL Structure
```
/api/v{n}/<resource>[/<id>][/<sub-resource>]
```

- All public endpoints live under `/api/v1/`.
- Resource names are **plural** and **kebab-case** (e.g. `/api/v1/reading-progress`).
- IDs are URL-safe ULIDs (preferred) or UUIDs.
- Versioning bumps (`/api/v2/...`) only when introducing breaking changes.

### Standard Resources

| Resource | Path | Description |
|----------|------|-------------|
| Manga catalog | `/api/v1/manga` | List, search, filter by genre/status |
| Manga detail | `/api/v1/manga/{id}` | Series metadata, latest chapters |
| Chapter list | `/api/v1/manga/{id}/chapters` | Chapters of a series |
| Chapter detail | `/api/v1/chapters/{id}` | Chapter metadata |
| Chapter pages | `/api/v1/chapters/{id}/pages` | Page manifest for the reader |
| Reading progress | `/api/v1/me/progress` | Per-user progress (read/unread) |
| Library | `/api/v1/me/library` | User favorites and shelves |
| Bookmarks | `/api/v1/me/bookmarks` | Page-level bookmarks |
| Search | `/api/v1/search?q=...` | Full-text and tag search |
| Auth | `/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/auth/logout` | Token issuance |
| Health | `/healthz`, `/readyz` | Liveness/readiness probes |
| Metrics | `/metrics` | Prometheus scrape endpoint |

### HTTP Methods
- `GET` — read resources (idempotent, cacheable)
- `POST` — create resources or perform actions (`/auth/login`)
- `PATCH` — partial updates
- `PUT` — full replacement (rare)
- `DELETE` — remove (soft-delete by default)

### Response Format

Success body:
```json
{
  "data": { ... },           // single resource
  "meta": { ... }            // optional (pagination, timing)
}
```

List body:
```json
{
  "data": [ ... ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 1234,
    "next_cursor": "..."
  }
}
```

Error body (see [04-error-handling.md](04-error-handling.md)):
```json
{
  "error": {
    "code": "CHAPTER_NOT_FOUND",
    "message": "chapter not found",
    "details": { "chapter_id": "..." }
  }
}
```

### Pagination
- Default `page_size` = 20, max = 100.
- Prefer **cursor pagination** for infinite-scroll catalog and reader history.
- Use **page-based** only for admin views or when the total count matters.

### Filtering and Sorting
- Filters as query params: `?genre=action&status=ongoing&language=ko`.
- Sorting: `?sort=-published_at,title` (`-` prefix = descending).
- Reject unknown sort fields with `400 BAD_REQUEST`.

### Caching Headers
- `GET` responses for static-ish resources MUST set `Cache-Control` and `ETag`.
- Page manifests: `Cache-Control: public, max-age=600` and a strong ETag derived from the chapter version.
- Per-user resources: `Cache-Control: private, no-store`.

## Handler Standards

### Handler Skeleton

```go
type ChapterHandler struct {
	service ChapterService
	logger  *logger.Logger
}

func (h *ChapterHandler) GetPages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chapterID := chi.URLParam(r, "id")

	if chapterID == "" {
		httpx.WriteError(w, NewValidationError(CodeValMissing, "chapter id is required"))
		return
	}

	manifest, err := h.service.GetPageManifest(ctx, chapterID)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, manifest)
}
```

### Rules for All Handlers

1. **Context-aware** — Always thread `r.Context()` through to services.
2. **No business logic** — handlers parse, validate, and translate.
3. **Validate at the edge** — reject malformed input *before* hitting the service.
4. **Status codes** — use the closest semantic match:
   - `200 OK` — successful read/update
   - `201 Created` — successful resource creation
   - `204 No Content` — successful delete
   - `400 Bad Request` — validation failure
   - `401 Unauthorized` — missing/invalid auth
   - `403 Forbidden` — authenticated but not allowed
   - `404 Not Found` — resource missing
   - `409 Conflict` — version/state conflict
   - `429 Too Many Requests` — rate limited
   - `500 Internal Server Error` — unexpected
5. **Never leak internals** — error responses go through the error mapper that strips stack traces in production.
6. **Set request IDs** — middleware injects `X-Request-Id`; include it in every log line.

## Authentication & Authorization

### Token Strategy

| Token Type | Use Case | Storage | TTL |
|------------|----------|---------|-----|
| Access (JWT) | API calls from web/mobile | Authorization header | 60 min |
| Refresh | Obtain new access token | HttpOnly cookie or secure storage | 30 days |
| Session ID | Browser SSR / cookie auth | HttpOnly cookie + Redis | 60 min sliding |

### JWT Claims

```json
{
  "iss": "steins",
  "sub": "user-id",
  "iat": 1714300000,
  "exp": 1714303600,
  "scope": ["read:catalog", "read:reader", "write:library"],
  "role": "reader"
}
```

### Roles

| Role | Permissions |
|------|-------------|
| `guest` | Read public catalog only |
| `reader` | All reader-side endpoints (default for registered users) |
| `uploader` | Add/edit chapters within assigned series |
| `admin` | Full access |

### Middleware Pattern

```go
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, NewUnauthorizedError(CodeAuthMissing, "authentication required"))
				return
			}
			if !claims.HasRole(role) {
				httpx.WriteError(w, NewForbiddenError(CodeAuthForbidden, "insufficient role"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

## Domain Models

### Manga

```go
type Manga struct {
	// Identity
	ID          string
	Slug        string

	// Content
	Title       string
	AltTitles   []string
	Description string
	CoverKey    string    // object storage key

	// Classification
	Authors     []string
	Artists     []string
	Genres      []string
	Tags        []string
	Status      Status    // ongoing | completed | hiatus | cancelled
	Language    string    // ISO 639-1
	ReadingDir  Direction // ltr | rtl | vertical

	// Metrics
	ChapterCount int
	ViewCount    int64
	Rating       float64
	RatingCount  int

	// Timestamps
	PublishedAt time.Time
	UpdatedAt   time.Time
	CreatedAt   time.Time
}
```

### Chapter

```go
type Chapter struct {
	ID         string
	MangaID    string

	Number     float64    // supports decimals (e.g. 12.5)
	Volume     *int
	Title      string
	Language   string

	PageCount  int
	Status     ChapterStatus // pending | processing | ready | failed
	Version    int           // increments on republish

	PublishedAt time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
```

### Page

```go
type Page struct {
	ID         string
	ChapterID  string
	Index      int        // 1-based page order

	OriginalKey   string  // object storage key for the original
	PreviewKey    string  // resized variant for the reader
	ThumbnailKey  string

	Width      int
	Height     int
	ByteSize   int64
	MimeType   string

	CreatedAt  time.Time
}
```

### User

```go
type User struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string         // empty for OAuth-only users
	Role         Role
	Locale       string
	Preferences  UserPreferences

	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserPreferences struct {
	ReadingDir         Direction
	ImageVariant       string  // thumbnail | preview | original
	NSFWVisible        bool
	PreferredLanguage  string
}
```

### Reading Progress

```go
type ReadingProgress struct {
	UserID     string
	MangaID    string
	ChapterID  string
	PageIndex  int     // last read page (1-based)
	ReadAt     time.Time
}
```

### Library Entry

```go
type LibraryEntry struct {
	UserID    string
	MangaID   string
	Shelf     string  // e.g. "reading", "completed", "plan-to-read"
	AddedAt   time.Time
	Rating    *int
	Notes     string
}
```

## Reader Endpoints

The reader is the core of the service — it MUST be fast, predictable, and resumable.

### Page Manifest

`GET /api/v1/chapters/{id}/pages`

Response:
```json
{
  "data": {
    "chapter_id": "01J...",
    "manga_id": "01J...",
    "version": 3,
    "reading_dir": "rtl",
    "pages": [
      {
        "index": 1,
        "url": "https://cdn.steins.example/.../001.webp",
        "width": 1280,
        "height": 1860
      },
      ...
    ]
  }
}
```

Rules:
- URLs are signed with a short TTL when content is gated; otherwise plain CDN URLs
- The manifest is cached at the edge — invalidated when `version` bumps
- `width`/`height` MUST be present so the client can lay out the page before bytes arrive

### Progress Updates

`PUT /api/v1/me/progress/{manga_id}`

```json
{ "chapter_id": "...", "page_index": 12 }
```

- Idempotent (same payload twice yields the same state)
- Writes through to Postgres; cached in Redis for 60 s
- `409 Conflict` if `chapter_id` does not belong to `manga_id`

### Bookmarks

`POST /api/v1/me/bookmarks` — create a bookmark at (chapter, page).
`DELETE /api/v1/me/bookmarks/{id}` — remove.

## Validation

### Request Validation

Use `github.com/go-playground/validator/v10` (or hand-rolled validators) at the handler boundary:

```go
type CreateLibraryEntryRequest struct {
	MangaID string `json:"manga_id" validate:"required,len=26"`
	Shelf   string `json:"shelf"    validate:"required,oneof=reading completed plan-to-read dropped"`
	Rating  *int   `json:"rating"   validate:"omitempty,min=1,max=10"`
}
```

### Validation Rules

| Field | Rule |
|-------|------|
| `email` | RFC 5322 + DNS check optional |
| `username` | 3–32 chars, `[a-z0-9_-]`, lowercase |
| `password` | ≥10 chars, mix of letter+digit (use Argon2id for hashing) |
| `slug` | 1–80 chars, kebab-case |
| `chapter_number` | `> 0`, max 4 decimal places |
| `page_index` | `>= 1`, `<= chapter.PageCount` |
| `rating` | integer 1–10 |
| `shelf` | enum |

### Validation Errors

Return all validation failures in a single response:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "request validation failed",
    "details": {
      "fields": [
        { "field": "email", "rule": "email", "message": "must be a valid email" },
        { "field": "password", "rule": "min", "message": "must be at least 10 characters" }
      ]
    }
  }
}
```

## Rate Limiting

### Defaults (per IP unless noted)

| Endpoint group | Limit | Window |
|----------------|-------|--------|
| `POST /auth/login` | 10 | 1 min |
| `POST /auth/register` | 5 | 1 hour |
| `GET /api/v1/manga`, `GET /api/v1/search` | 120 | 1 min |
| `GET /api/v1/chapters/{id}/pages` | 600 | 1 min |
| `PUT /api/v1/me/progress/*` | 600 | 1 min (per user) |
| Anonymous global | 60 | 1 min |

### Implementation

- Token bucket in Redis using `INCR` + `EXPIRE`
- Key format: `rate:{group}:{ip_or_user}`
- 429 response includes `Retry-After` and `X-RateLimit-Remaining` headers

```go
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error)
}
```

## Image Delivery

### Variants

| Variant | Width | Format | Use |
|---------|-------|--------|-----|
| `thumbnail` | 256px | WebP | catalog cards, hover previews |
| `preview` | 640px | WebP | mobile reader default |
| `web` | 1280px | WebP | desktop reader default |
| `original` | n/a | original format | downloads, premium readers |

- Stored as `<chapter_id>/<page_index>.<variant>.<ext>` in object storage
- The API never streams image bytes itself — it returns CDN URLs

### Signed URLs

For gated content (paid chapters, age-gated, draft uploads):
- API generates a presigned URL with TTL = 600 s
- TTL is configurable via `storage.signed_url_ttl_s`
- The CDN respects the underlying object storage signature

### Pre-fetching

- Reader clients SHOULD pre-fetch the next 3 pages
- The page manifest hints this with `"prefetch_next": 3`

## Search

### Endpoint
`GET /api/v1/search?q=<query>&genre=...&language=...&page=1&page_size=20`

### Behavior
- Backed by Meilisearch index `manga`
- Falls back to PostgreSQL `tsvector` when Meilisearch is unavailable
- Returns up to 100 results per request
- Highlights matched terms in `title` and `description`

### Search Document

```go
type MangaSearchDoc struct {
	ID         string   `json:"id"`
	Slug       string   `json:"slug"`
	Title      string   `json:"title"`
	AltTitles  []string `json:"alt_titles"`
	Authors    []string `json:"authors"`
	Genres     []string `json:"genres"`
	Tags       []string `json:"tags"`
	Status     string   `json:"status"`
	Language   string   `json:"language"`
	UpdatedAt  int64    `json:"updated_at"`   // unix
	Popularity float64  `json:"popularity"`   // ranking signal
}
```

## Repository Pattern

### Interface

Define repository interfaces in the domain package, implementations in `internal/storage/...`:

```go
// internal/chapter/repository.go
type Repository interface {
	FindByID(ctx context.Context, id string) (*Chapter, error)
	ListByManga(ctx context.Context, mangaID string, opts ListOpts) ([]*Chapter, error)
	Insert(ctx context.Context, chapter *Chapter) error
	UpdateStatus(ctx context.Context, id string, status ChapterStatus, version int) error
	IncrementViews(ctx context.Context, id string) error
}
```

### Postgres Implementation

```go
// internal/storage/postgres/chapter_repository.go
type ChapterRepository struct {
	pool *pgxpool.Pool
}

func (r *ChapterRepository) FindByID(ctx context.Context, id string) (*chapter.Chapter, error) {
	const q = `
SELECT id, manga_id, number, volume, title, language, page_count, status, version,
       published_at, created_at, updated_at
FROM chapters
WHERE id = $1
`
	var c chapter.Chapter
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&c.ID, &c.MangaID, &c.Number, &c.Volume, &c.Title, &c.Language,
		&c.PageCount, &c.Status, &c.Version,
		&c.PublishedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("query chapter %s: %w", id, err)
	}
	return &c, nil
}
```

### Rules

- Repositories return domain models, not raw rows.
- Repositories never log — they return wrapped errors and let the service decide.
- Use prepared statements / `pgx` parameter binding — never string concatenation.
- Wrap `pgx.ErrNoRows` as `storage.ErrNotFound`.
- Transactions are owned by the **service** layer via `pool.BeginTx`.

## Service Layer Patterns

### Transaction Boundary

Services own transactions. A service method that touches multiple repositories opens a single transaction and passes it down:

```go
func (s *ChapterService) Publish(ctx context.Context, id string) error {
	return s.tx.Run(ctx, func(tx storage.Tx) error {
		ch, err := s.chapters.WithTx(tx).FindByID(ctx, id)
		if err != nil {
			return err
		}
		ch.Status = ChapterStatusReady
		ch.Version++
		if err := s.chapters.WithTx(tx).UpdateStatus(ctx, ch.ID, ch.Status, ch.Version); err != nil {
			return err
		}
		return s.queue.Enqueue(ctx, "indexing", IndexChapterPayload{ID: ch.ID})
	})
}
```

### Idempotency

Mutating endpoints SHOULD accept an `Idempotency-Key` header for retries:

- Hash the key + user + endpoint
- Store the response in Redis with 24h TTL
- Return the cached response on retry

## Health & Readiness

### Endpoints
- `GET /healthz` — process is alive (always 200 if reachable)
- `GET /readyz` — checks Postgres, Redis, object storage, queue

```go
func (s *ReadinessChecker) Check(ctx context.Context) Health {
	checks := []HealthCheck{
		s.db.Check(ctx),
		s.redis.Check(ctx),
		s.objects.Check(ctx),
		s.queue.Check(ctx),
	}
	return aggregate(checks)
}
```

A failed `readyz` removes the instance from the load balancer; `healthz` keeps it alive so it can recover.

## Observability

### Required Logs

Every handler MUST log on completion:
```go
log.WithFields(map[string]interface{}{
	"request_id":  requestID,
	"method":      r.Method,
	"path":        r.URL.Path,
	"status":      status,
	"duration_ms": elapsed.Milliseconds(),
	"user_id":     userID, // if authenticated
}).Info("request completed")
```

### Required Metrics

```go
httpRequests = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "steins_http_requests_total",
		Help: "Total HTTP requests",
	},
	[]string{"method", "route", "status"},
)

httpDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "steins_http_request_duration_seconds",
		Help:    "HTTP request duration",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"method", "route"},
)

readerPagesServed = promauto.NewCounter(
	prometheus.CounterOpts{
		Name: "steins_reader_pages_served_total",
		Help: "Pages returned via the reader manifest",
	},
)
```

## Code Organization Per Domain

```
internal/chapter/
├── service.go         # ChapterService implementation
├── service_iface.go   # Interface used by handlers
├── repository.go      # Repository interface
├── model.go           # Domain types
├── errors.go          # Domain-specific error codes
└── doc.go             # Package doc

internal/api/handler/chapter/
├── handler.go         # ChapterHandler
├── routes.go          # Route registration
└── dto.go             # Request/response DTOs
```

### Common Patterns

1. **Constructor functions** for all services and handlers (`NewChapterService`)
2. **Interface-driven** boundaries — handlers depend on a `ChapterService` interface
3. **DTO ↔ domain** translation isolated in `dto.go`
4. **Repository tests** use real Postgres (testcontainers), never mocks
5. **Service tests** mock repositories
