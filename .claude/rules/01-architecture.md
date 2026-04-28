# Architecture and Design Principles

## Core Architecture

### System Overview
- **Purpose**: Web-based manga reading platform
- **Initial Scope**: Catalog browsing, chapter reading, user library, search
- **Design Philosophy**: Reader-first UX, scalable image delivery, maintainable

### Architectural Layers

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
│  (Interfaces, domain models)            │
├─────────────────────────────────────────┤
│     Storage Layer                       │
│  (PostgreSQL, Redis, Object Storage)    │
└─────────────────────────────────────────┘
```

## Design Principles

### 1. Modularity
- Each domain (manga, user, library, reader) MUST be a separate package
- Use interface-based design at service and repository boundaries
- Storage backends are pluggable (Postgres now, others later)

### 2. Scalability
- Stateless API servers — horizontally scalable behind a load balancer
- Background workers scale independently of the API
- Heavy traffic paths (page reads, list queries) cached aggressively
- Image delivery offloaded to CDN / object storage signed URLs

### 3. Extensibility
- Configuration-driven feature toggles
- Pluggable storage and search backends
- Adding new image formats or reader modes should not touch the API contract

### 4. Data Integrity
- Validate at the API boundary AND at the domain boundary
- Treat object storage as the source of truth for image bytes
- Treat PostgreSQL as the source of truth for metadata
- Maintain referential integrity (chapter → manga, page → chapter)

## Current Implementation Status

### 📋 Planned (v0.1.0 – initial release)
- **HTTP API server** with chi/echo router
- **PostgreSQL** for relational data (manga, chapters, pages, users, libraries)
- **Redis** for sessions, rate limiting, hot caches
- **Object storage** (S3-compatible) for image bytes
- **Asynq** background workers for image processing and indexing
- **Meilisearch** (or PostgreSQL FTS as starter) for search
- **Structured logging** with zerolog
- **Prometheus metrics** and `/healthz` endpoint
- **Standard Go project layout**, Makefile, multiple binaries

### 🔮 Later
- WebP/AVIF derivative generation
- Recommendation engine (collaborative filtering)
- OAuth2 social login
- Reader analytics dashboard
- Mobile-app friendly endpoints (cursor pagination, ETags)

## Directory Structure

Following [Standard Go Project Layout](https://github.com/golang-standards/project-layout):

```
steins/
├── cmd/                        # Application entry points (all build to bin/)
│   ├── api/                    # HTTP API server → bin/api
│   │   └── main.go
│   ├── worker/                 # Background job worker → bin/worker
│   │   └── main.go
│   ├── steins/                 # API + worker combined → bin/steins
│   │   └── main.go
│   ├── migrate/                # DB migration (up) → bin/migrate
│   │   └── main.go
│   └── migrate-down/           # DB migration (down) → bin/migrate-down
│       └── main.go
│
├── internal/                   # Private application code
│   ├── api/
│   │   ├── handler/            # HTTP handlers (manga, chapter, reader, user)
│   │   ├── middleware/         # Auth, logging, rate limit, CORS
│   │   └── server.go           # Router wiring
│   ├── manga/                  # Manga domain
│   │   ├── service.go          # Application service
│   │   ├── repository.go       # Repository interface
│   │   └── model.go            # Domain types
│   ├── chapter/                # Chapter & page domain
│   ├── reader/                 # Reading progress, bookmarks
│   ├── user/                   # Account, profile, auth
│   ├── library/                # Per-user library, favorites, history
│   ├── search/                 # Search service (Meilisearch / FTS)
│   ├── image/                  # Image processing (resize, encode)
│   ├── worker/                 # Asynq job handlers
│   └── storage/
│       ├── postgres/           # PostgreSQL repository implementations
│       ├── redis/              # Redis-backed implementations
│       └── object/             # S3-compatible object storage client
│
├── pkg/                        # Public library code
│   ├── logger/                 # Reusable logger package (zerolog wrapper)
│   ├── httpx/                  # HTTP helpers (response, error mapping)
│   ├── config/                 # Configuration loader
│   └── auth/                   # JWT / token helpers
│
├── test/                       # Test files (mirrors service architecture)
│   ├── internal/               # tests for internal/ packages
│   │   ├── api_handler/        # ← internal/api/handler/
│   │   ├── manga/              # ← internal/manga/
│   │   ├── chapter/            # ← internal/chapter/
│   │   └── storage_postgres/   # ← internal/storage/postgres/
│   └── pkg/                    # tests for pkg/ packages
│       ├── config/
│       └── logger/
│
├── examples/                   # Usage examples
│   └── basic_usage.go
│
├── migrations/                 # SQL migrations (golang-migrate format)
├── configs/                    # Configuration files
├── scripts/                    # Build and deployment scripts
├── deployments/
│   └── docker/
│
├── web/                        # Static assets and (optional) templates
│   ├── static/                 # CSS, JS bundles, icons
│   └── templates/              # HTML templates if SSR is used
│
├── docs/
│   ├── en/                     # English docs
│   └── ko/                     # Korean docs
│
├── .claude/                    # Claude AI development rules
│   └── rules/
│
├── Makefile                    # Build automation
├── go.mod                      # Go module definition
├── go.sum                      # Dependency checksums
└── README.md                   # Project documentation
```

### Directory Purposes

**`cmd/`**: Application entry points (main packages)
- Each subdirectory is an executable; all build to `bin/<name>` via `make build`
- Minimal logic, imports from `internal/` and `pkg/`
- New entry points MUST be added to the `build` target in `Makefile` alongside the corresponding `*_BINARY` variable

**`internal/`**: Private application code
- Cannot be imported by external projects
- Core business logic
- Domain-specific implementations

**`pkg/`**: Public library code
- Can be imported by external projects
- Reusable, generic utilities
- Well-documented, production-ready

**`test/`**: Test files
- 모든 테스트 파일은 소스 코드와 분리하여 `test/` 아래에만 위치
- **서비스 아키텍처와 동일한 디렉토리 구조**를 따름
  - `internal/<pkg>/` → `test/internal/<pkg>/`
  - `pkg/<pkg>/` → `test/pkg/<pkg>/`
- 패키지 선언은 `package <name>_test` (외부 테스트 패키지)

## Technology Stack

### Core
- **Language**: Go 1.22+
- **HTTP Router**: `github.com/go-chi/chi/v5` (lightweight, idiomatic)
- **HTTP Client**: Custom client with connection pooling (used for object storage / external integrations)
- **Logging**: Structured logging with `github.com/rs/zerolog`
- **Testing**: `testing` + `github.com/stretchr/testify`

### Storage
- **Primary DB**: PostgreSQL 15+
  - Manga, chapters, pages, users, libraries, sessions
  - Connection pooling via `pgx/v5`
- **Cache**: Redis 7+
  - Sessions, rate limit counters, hot resource cache
- **Object Storage**: S3-compatible (AWS S3, MinIO, Cloudflare R2)
  - Original chapter images
  - Generated thumbnails / web variants
- **Search**: Meilisearch (or PostgreSQL `tsvector` for v0.1.0)

### Background Jobs
- **Queue**: `github.com/hibiken/asynq` (Redis-backed)
- **Use Cases**: Image processing, search indexing, e-mail, recommendation refresh
- **Why not Kafka**: Throughput requirements are modest; Asynq gives retries, scheduling, and a UI without operational overhead

### Authentication
- **JWT** for stateless API tokens
- **Session cookies** (Redis-backed) for browser clients
- **OAuth2** (planned) — Google/Discord/Apple via `golang.org/x/oauth2`

### Image Processing
- **Library**: `github.com/disintegration/imaging` (resizing) + `github.com/chai2010/webp` for WebP encoding
- **Targets**: thumbnail (256px wide), preview (640px), full (original)

### Observability
- **Metrics**: Prometheus (`/metrics` endpoint)
- **Tracing**: OpenTelemetry (planned)
- **Monitoring**: Grafana dashboards

### Dependencies (target)
```go
require (
  github.com/go-chi/chi/v5 v5.x
  github.com/jackc/pgx/v5 v5.x
  github.com/redis/go-redis/v9 v9.x
  github.com/hibiken/asynq v0.x
  github.com/golang-jwt/jwt/v5 v5.x
  github.com/rs/zerolog v1.x
  github.com/stretchr/testify v1.x
)
```

## Data Flow

### Read Path (most common)

```
Reader → CDN (cache miss?) → API server →
  Redis (cache miss?) → PostgreSQL → response
                                   ↘ object storage signed URL
```

1. Browser requests `/api/v1/chapters/{id}/pages`
2. Edge CDN serves the cached JSON if fresh; otherwise hits the API
3. API checks Redis for the chapter manifest; on miss it queries PostgreSQL
4. Page records contain object storage keys; the API returns either signed URLs or proxy paths
5. The browser fetches images directly from the CDN-fronted object storage

### Write Path (chapter upload)

```
Uploader → API → object storage (originals) →
  Postgres (chapter pending) → Asynq enqueue →
    worker → image processing → Postgres (chapter ready) →
      worker → search indexer
```

1. An authenticated uploader (or admin) POSTs a chapter with image files
2. The API streams originals into object storage and writes a pending row in `chapters`
3. The API enqueues a `process_chapter` job
4. The worker generates derivatives (thumbnail, web variant), updates page records, and flips the chapter status to `ready`
5. Another job updates the search index and refreshes recommendation signals

### Background Job Topics (Asynq queues)

| Queue | Purpose | Priority |
|-------|---------|----------|
| `images` | Resize / encode chapter pages | high |
| `indexing` | Push manga/chapter docs to search | normal |
| `notifications` | E-mail / push | normal |
| `recommendations` | Refresh similarity scores | low |
| `cleanup` | Purge expired sessions, orphaned files | low |

## Configuration Strategy

### Multi-Environment Support
- Development, Staging, Production configs
- Environment variable overrides
- Secrets management (Vault, AWS Secrets Manager, or `.env` for local)

### Sample Configuration

```yaml
server:
  http_addr: ":8080"
  read_timeout_ms: 5000
  write_timeout_ms: 10000

database:
  url: "postgres://steins:steins@localhost:5432/steins?sslmode=disable"
  max_open_conns: 25
  max_idle_conns: 5

redis:
  addr: "localhost:6379"
  db: 0

storage:
  provider: "s3"      # s3 | minio | r2
  bucket: "steins-content"
  region: "ap-northeast-2"
  signed_url_ttl_s: 600

search:
  provider: "meilisearch"
  url: "http://localhost:7700"

asynq:
  redis_addr: "localhost:6379"
  concurrency: 20
  queues:
    images: 6
    indexing: 3
    notifications: 2
    recommendations: 1
    cleanup: 1

reader:
  default_image_variant: "preview"   # thumbnail | preview | original
  max_pages_per_chapter: 500

auth:
  jwt_issuer: "steins"
  jwt_ttl_min: 60
  refresh_ttl_day: 30
```

## Scalability Considerations

### Horizontal Scaling

1. **API Layer**
   - Fully stateless — every request carries its own auth token / session ID
   - Run N replicas behind a load balancer
   - Sticky sessions are NOT required

2. **Worker Layer**
   - Asynq consumers across multiple processes pick up work concurrently
   - Scale per-queue concurrency based on backlog
   - Heavy queues (images) can run on dedicated nodes

3. **Database**
   - PostgreSQL with read replicas for heavy read paths (catalog, search)
   - Use a connection pooler (pgbouncer) under load
   - Partition `pages` table by chapter creation month if it grows past ~50M rows

4. **Object Storage**
   - Effectively infinite — fronted by a CDN
   - Use lifecycle rules to expire stale derivative variants

### Performance Targets

1. **Throughput**
   - 1,000+ concurrent readers per API instance
   - 200+ chapter uploads/hour processed by the worker pool
   - Reader page-list endpoint: p95 < 100ms (warm cache), < 400ms (cold)

2. **Availability**
   - 99.9% uptime for the read path
   - Read path SHOULD remain available during DB primary failover (read-replica fallback)
   - Image delivery available even if the API is down (CDN cached)

3. **Resource Targets**
   - API instance: 256MB RAM baseline, 512MB peak
   - Worker instance: 512MB RAM baseline, 1GB peak (image jobs)
   - Postgres: tuned for ~80% buffer cache hit ratio

### Caching Strategy

| Resource | Layer | TTL | Invalidation |
|----------|-------|-----|--------------|
| Manga detail | Redis | 5 min | On manga update |
| Chapter list (per manga) | Redis | 1 min | On chapter add/update |
| Page manifest | Redis | 10 min | On chapter republish |
| Search results | Redis | 30 s | TTL-only |
| User session | Redis | 60 min sliding | On logout |
| Static images | CDN | 30 days | Versioned URLs |
| Reading progress | Postgres + Redis cache | n/a | Write-through |
