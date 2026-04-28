# Error Handling and Monitoring Rules

## Error Handling Principles

### Core Principles

1. **Fail Gracefully**
   - NEVER panic in production code
   - Use `recover()` only in top-level handlers (HTTP middleware, job workers)
   - Return errors, don't swallow them
   - Log errors with full context

2. **Error Context**
   - Wrap errors with context using `fmt.Errorf` with `%w`
   - Include relevant information (resource ID, user ID, route, queue)
   - Preserve error chain for debugging

3. **Error Types**
   - Use custom error types for different categories
   - Enable type-based handling and HTTP status mapping
   - Include machine-readable error codes

## Layer Rules — When to Use `AppError` vs `fmt.Errorf`

본 프로젝트는 `internal/apperr.AppError` 를 시스템 전반의 구조화 에러 타입으로 사용합니다.
모든 에러 생성을 일률적으로 구조화하지 않고, 레이어별 규칙을 명확히 분리합니다.

### MUST — `AppError` 로 반환해야 하는 boundary

다음 위치에서 발생하는 에러는 반드시 `apperr.NewXxxError(...)` 생성자로 만들어 반환합니다.
HTTP 상태 매핑, 로깅 라우팅, 재시도 전략이 카테고리·코드에 의존하기 때문입니다.

| 레이어 | 카테고리 | 생성자 |
|---|---|---|
| `internal/api/handler/` 의 검증 결과 | `Validation`, `Auth`, `NotFound`, `Conflict`, `RateLimit` | `NewValidationError`, `NewUnauthorizedError`, `NewForbiddenError`, `NewNotFoundError`, `NewConflictError`, `NewRateLimitError` |
| `internal/<domain>/service.go` 의 boundary 메소드 | `Validation`, `NotFound`, `Conflict`, `Forbidden` | 도메인별 코드 (`CHAPTER_NOT_FOUND`, `MANGA_LOCKED`, ...) |
| `internal/storage/` 의 boundary | `Storage`, `Database` | `NewStorageError` (`STORAGE_001` 조회 / `STORAGE_002` 저장 / `STORAGE_003` 삭제), `NewDatabaseError` |
| `internal/worker/` 의 작업 결과 | `Worker`, `Image`, `Search` | `NewWorkerError`, `NewImageError`, `NewSearchError` |
| 외부 시스템 호출 (object storage, search) | `Network`, `Timeout`, `RateLimit` | `NewNetworkError`, `NewTimeoutError`, `NewRateLimitError` |

### MAY — `fmt.Errorf` 유지가 적절한 레이어

다음 코드는 `fmt.Errorf("동사 대상 [식별자]: %w", ...)` 패턴을 유지합니다.

- **`pkg/` 의 generic 유틸 패키지** (`pkg/httpx`, `pkg/config`, `pkg/auth`, `pkg/logger`)
  - `pkg/` 는 도메인 의존성을 갖지 않는 standalone 라이브러리로 설계됨.
  - `internal/apperr` 를 import 하면 레이어링이 무너짐.
  - 호출하는 `internal/` boundary 에서 카테고리에 맞는 `AppError` 로 변환해 wrap 합니다.
- **`internal/` 의 helper / 내부 함수**
  - 패키지 외부로 노출되지 않는 helper 는 fmt.Errorf 로 충분.
  - 외부 노출 함수에서 최종 boundary 변환만 보장하면 됩니다.

### 예시 — boundary 변환 패턴

```go
// pkg/httpx/decode.go (generic util — fmt.Errorf 유지)
func DecodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("decode json body: %w", err)
	}
	return nil
}

// internal/api/handler/chapter/handler.go (boundary — AppError 로 변환)
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateChapterRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, apperr.NewValidationError(
			apperr.CodeValBadJSON,
			"invalid request body",
			err,
		))
		return
	}
	// ...
}
```

### `AppError.Is` 와 `errors.As`

`AppError.Is(target)` 는 `Code` 또는 `Category` 가 동일하면 true.
호출자는 다음 두 가지 패턴 중 적합한 쪽을 사용합니다.

```go
// 패턴 A — 카테고리 분기
var ae *apperr.AppError
if errors.As(err, &ae) && ae.Category == apperr.ErrCategoryRateLimit {
	// backoff 길게
}

// 패턴 B — 특정 코드 매칭 (errors.Is 는 AppError.Is 를 호출)
if errors.Is(err, &apperr.AppError{Code: apperr.CodeChapterNotFound}) {
	// not-found 전용 처리
}
```

## Error Taxonomy

### Error Categories

```go
type ErrorCategory string

const (
	// Client-caused — return 4xx
	ErrCategoryValidation   ErrorCategory = "validation"
	ErrCategoryAuth         ErrorCategory = "auth"
	ErrCategoryForbidden    ErrorCategory = "forbidden"
	ErrCategoryNotFound     ErrorCategory = "not_found"
	ErrCategoryConflict     ErrorCategory = "conflict"
	ErrCategoryRateLimit    ErrorCategory = "rate_limit"

	// Transient — retryable
	ErrCategoryNetwork      ErrorCategory = "network"
	ErrCategoryTimeout      ErrorCategory = "timeout"

	// System — return 5xx
	ErrCategoryDatabase     ErrorCategory = "database"
	ErrCategoryStorage      ErrorCategory = "storage"
	ErrCategoryQueue        ErrorCategory = "queue"
	ErrCategorySearch       ErrorCategory = "search"
	ErrCategoryImage        ErrorCategory = "image"

	// Logic — return 5xx
	ErrCategoryConfig       ErrorCategory = "config"
	ErrCategoryInternal     ErrorCategory = "internal"
)
```

### Custom Error Type

```go
type AppError struct {
	Category   ErrorCategory
	Code       string
	Message    string
	Resource   string         // resource type (e.g. "chapter")
	ResourceID string         // resource id when applicable
	StatusCode int            // HTTP status to surface
	Retryable  bool
	Details    map[string]any // safe-to-expose details
	Err        error          // wrapped error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s:%s] %s: %v", e.Category, e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s:%s] %s", e.Category, e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code || e.Category == t.Category
}
```

### Error Codes

Use consistent error codes across the system:

```
# Validation
VAL_001: missing required field
VAL_002: invalid field format
VAL_003: value out of range
VAL_004: unsupported enum value
VAL_005: invalid JSON body
VAL_006: file too large
VAL_007: unsupported MIME type

# Auth
AUTH_001: missing authentication
AUTH_002: invalid token
AUTH_003: token expired
AUTH_004: insufficient role
AUTH_005: account disabled

# Resources
MANGA_NOT_FOUND
MANGA_SLUG_TAKEN
CHAPTER_NOT_FOUND
CHAPTER_NUMBER_TAKEN
CHAPTER_PROCESSING
PAGE_NOT_FOUND
USER_NOT_FOUND
USER_EMAIL_TAKEN
LIBRARY_ENTRY_EXISTS

# Network / external
NET_001: connection refused
NET_002: connection timeout
NET_003: DNS resolution failed
HTTP_502: bad gateway
HTTP_503: service unavailable

# Storage
STORAGE_001: read failure
STORAGE_002: write failure
STORAGE_003: delete failure
STORAGE_004: object not found

# Database
DB_001: connection failed
DB_002: query timeout
DB_003: constraint violation
DB_004: deadlock detected

# Queue / worker
QUEUE_001: enqueue failed
QUEUE_002: payload malformed
WORKER_001: handler timeout
WORKER_002: payload not retryable

# Image
IMG_001: decode failed
IMG_002: encode failed
IMG_003: unsupported format
IMG_004: dimensions out of range

# Search
SEARCH_001: index unreachable
SEARCH_002: index write failed
```

## Retry Logic

### Retry Strategy

```go
type RetryPolicy struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	Jitter          bool
	RetryableErrors []ErrorCategory
}

var (
	NetworkRetryPolicy = RetryPolicy{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableErrors: []ErrorCategory{
			ErrCategoryNetwork,
			ErrCategoryTimeout,
		},
	}

	RateLimitRetryPolicy = RetryPolicy{
		MaxAttempts:  5,
		InitialDelay: 10 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		Jitter:       true,
		RetryableErrors: []ErrorCategory{
			ErrCategoryRateLimit,
		},
	}
)
```

### Retry Implementation

```go
func WithRetry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < policy.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateBackoff(policy, attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		var appErr *AppError
		if errors.As(err, &appErr) && !appErr.Retryable {
			return err
		}

		lastErr = err
		log.WithFields(map[string]interface{}{
			"attempt":      attempt + 1,
			"max_attempts": policy.MaxAttempts,
		}).WithError(err).Warn("retrying after error")
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Circuit Breaker

Implement circuit breaker for external services:

```go
type CircuitBreaker struct {
	MaxFailures     int
	Timeout         time.Duration
	ResetTimeout    time.Duration

	state           State  // Closed, Open, HalfOpen
	failures        int
	lastFailureTime time.Time
	mu              sync.RWMutex
}
```

Apply circuit breakers to:
- Object storage operations under sustained failure
- Meilisearch / search backend calls
- Outgoing third-party calls (e-mail provider, OAuth token endpoint)
- Database connections (via the pool's own protections + a shallow breaker for repeated timeouts)

## HTTP Error Mapping

The HTTP layer translates `AppError.Category` into status codes:

| Category | HTTP Status |
|----------|-------------|
| `validation` | 400 |
| `auth` | 401 |
| `forbidden` | 403 |
| `not_found` | 404 |
| `conflict` | 409 |
| `rate_limit` | 429 |
| `timeout` | 504 (server-initiated) or 408 (client) |
| `network`, `database`, `storage`, `queue`, `search`, `image`, `internal` | 500 |

```go
func WriteError(w http.ResponseWriter, err error) {
	var ae *AppError
	if !errors.As(err, &ae) {
		ae = &AppError{
			Category:   ErrCategoryInternal,
			Code:       "INTERNAL_ERROR",
			Message:    "unexpected error",
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusFor(ae))

	body := map[string]any{
		"error": map[string]any{
			"code":    ae.Code,
			"message": ae.Message,
			"details": ae.Details,
		},
	}
	_ = json.NewEncoder(w).Encode(body)
}
```

The handler MUST never expose `Err.Error()` directly to the client — wrap it inside `details` only when it's safe (validation paths) and strip it in production.

## Logging

### Structured Logging

Use `pkg/logger` (zerolog 기반 wrapper)로 구조화된 로깅을 수행합니다.
로그 메시지는 **반드시 영어**로 작성합니다. (주석·문서는 한국어 허용, 로그 msg 문자열은 영어)

```go
// Good — pkg/logger wrapper API
log.WithFields(map[string]interface{}{
	"route":       "/api/v1/chapters/{id}/pages",
	"chapter_id":  chapterID,
	"status_code": http.StatusOK,
	"duration_ms": elapsed.Milliseconds(),
}).Info("chapter pages served")

log.WithFields(map[string]interface{}{
	"chapter_id": chapterID,
	"error_code": "IMG_001",
}).WithError(err).Error("failed to decode page image")

// Bad — 로그 메시지에 한국어 사용 금지
log.WithError(err).Error("이미지 처리 실패")
```

### Log Levels

레벨 선택 기준:

| Level | 사용 시점 | 예시 시나리오 |
|-------|-----------|--------------|
| **DEBUG** | 개발 또는 트러블슈팅 시에만 필요한 내부 상세 정보 | 페이지 매니페스트 조립 단계, 캐시 hit/miss, 재시도 결정 |
| **INFO**  | 운영자가 프로덕션에서 확인하고 싶은 정상 동작 마일스톤 | 서버 기동, 챕터 처리 완료, 정기 cleanup 완료 |
| **WARN**  | 예상치 못했지만 gracefully 처리된 상황 — 시스템이 계속 동작 | 재시도 시도, search 백엔드 fallback 전환, shutdown timeout |
| **ERROR** | 작업 실패 — 주의가 필요하지만 다른 요청은 계속 처리 가능 | 핸들러 5xx 응답, 챕터 처리 실패, 인덱스 업데이트 실패 |
| **FATAL** | 복구 불가능한 오류 — 프로세스 종료 필요 | 설정 파싱 실패, 기동 시 필수 DB 연결 불가 |

**빠른 판단 기준:**
- 운영자가 알림을 받아야 하면: **ERROR** 또는 **FATAL**
- 처리됐지만 운영자가 알아야 하면: **WARN**
- 정상 동작 확인: **INFO**
- 개발자 트레이싱 용: **DEBUG**

```go
// DEBUG — 운영 환경에서는 필터링됨
log.WithField("chapter_id", id).Debug("assembling page manifest")
log.WithField("attempt", attempt).Debug("error not retryable, skipping retry")

// INFO — 정상 완료 마일스톤
log.WithFields(map[string]interface{}{"chapter_id": id, "version": v}).Info("chapter processing completed")
log.WithFields(map[string]interface{}{"deleted_count": n}).Info("orphan purge completed")

// WARN — 예상 외 상황이지만 처리됨
log.WithFields(map[string]interface{}{"attempt": a, "max_attempts": m, "delay_ms": d}).Warn("retrying after error")
log.WithError(err).Warn("meilisearch unavailable, falling back to postgres FTS")

// ERROR — 작업 실패
log.WithFields(map[string]interface{}{"chapter_id": id}).WithError(err).Error("chapter processing failed")
log.WithError(err).Error("failed to enqueue indexing job")
```

### Field Naming Conventions

모든 구조화 필드 키는 **snake_case**를 사용합니다. 아래 표에 정의된 표준 이름을 반드시 사용합니다.

| Field Key        | Type     | 설명 |
|-----------------|----------|------|
| `request_id`    | string   | 인바운드 요청 추적 ID |
| `route`         | string   | HTTP route pattern (예: `/api/v1/chapters/{id}`) |
| `method`        | string   | HTTP method |
| `status_code`   | int      | HTTP 응답 상태 코드 |
| `duration_ms`   | int64    | 경과 시간 (밀리초 단위) |
| `user_id`       | string   | 인증된 사용자 ID |
| `manga_id`      | string   | 만화 시리즈 ID |
| `chapter_id`    | string   | 챕터 ID |
| `page_index`    | int      | 페이지 순번 (1-based) |
| `version`       | int      | 챕터 republish 버전 |
| `error_code`    | string   | 에러 코드 상수 (예: `"VAL_001"`, `"IMG_001"`) |
| `attempt`       | int      | 현재 시도 횟수 (1-based) |
| `max_attempts`  | int      | 최대 시도 횟수 |
| `delay_ms`      | int64    | 재시도 backoff 대기 시간 (밀리초) |
| `task_type`     | string   | Asynq task 타입 (예: `"process_chapter"`) |
| `task_id`       | string   | Asynq task ID |
| `queue`         | string   | Asynq queue 이름 (예: `"images"`) |
| `worker_count`  | int      | 워커 goroutine 수 |
| `existing_id`   | string   | 중복 감지 시 기존 레코드 ID |
| `deleted_count` | int64    | 일괄 삭제된 레코드 수 |
| `host`          | string   | DB/Redis 호스트 |
| `port`          | int      | DB/Redis 포트 |
| `database`      | string   | DB 이름 |
| `max_conns`     | int32    | 커넥션 풀 최대 크기 |

**규칙:**
- 공통 표준 키 테이블에 정의된 키를 우선 사용하고, 표에 없는 컴포넌트 특화 키는 snake_case로 추가 가능 (예: `image_variant`, `bytes_in`, `bytes_out`)
- 이미 정의된 키와 의미가 중복되는 임의 키 생성 금지 (예: `"db_name"` 대신 `"database"` 사용)
- 시간 값은 반드시 밀리초 int64로, 키는 `duration_ms` 또는 `delay_ms`
- 불리언 플래그는 긍정형 이름 사용 (예: `"retryable"`, `"is_duplicate"`, `"signed_url"`)

### Per-Component Required Fields

컴포넌트별로 모든 로그 항목에 반드시 포함해야 하는 최소 필드입니다.

**HTTP middleware** (`internal/api/middleware/`):
- 모든 요청 로그: `request_id`, `method`, `route`
- 요청 완료 시: `status_code`, `duration_ms`
- 인증된 요청: `user_id`

**API handlers** (`internal/api/handler/`):
- 도메인 작업 로그: `request_id`, 관련 도메인 ID (예: `chapter_id`)
- 에러 발생 시: `error_code`

**Workers** (`internal/worker/`):
- 작업 로그: `task_type`, `task_id`, `queue`
- 작업 완료/실패: `duration_ms`, 성공시 결과 카운트
- 재시도: `attempt`, `max_attempts`, `delay_ms`

**Storage** (`internal/storage/postgres/`):
- 연결 성공: `host`, `port`, `database`, `max_conns`
- 트랜잭션 에러: `error_code`

**Image processor** (`internal/image/`):
- 페이지 처리 로그: `chapter_id`, `page_index`, `image_variant`
- 인코딩 통계: `bytes_in`, `bytes_out`, `duration_ms`

### Log Context

컴포넌트 초기화 시 scoped logger를 생성하여 반복 필드 중복을 방지합니다:

```go
type LogContext struct {
	RequestID string
	UserID    string
	MangaID   string
	ChapterID string
	TaskType  string
	TaskID    string
}

// buildLogger는 scoped logger를 생성하여 context에 저장합니다.
func buildLogger(ctx context.Context, lc LogContext) context.Context {
	log := logger.FromContext(ctx).WithFields(map[string]interface{}{
		"request_id": lc.RequestID,
		"user_id":    lc.UserID,
		"task_type":  lc.TaskType,
	})
	return log.ToContext(ctx)
}
```

### Log Storage

1. **Console**: 개발 및 디버깅 (Pretty mode)
2. **File**: 구조화된 JSON 로그
   - 일별 로테이션
   - 오래된 로그 압축
   - 30일 보존
3. **Centralized**: 프로덕션 로깅
   - Loki, Elasticsearch, 또는 CloudWatch 사용
   - 전문 검색 활성화
   - 에러 패턴 알림 설정

## Monitoring

### Metrics Collection

Use Prometheus for metrics:

```go
var (
	// HTTP metrics
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
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)

	// Reader metrics
	pagesServed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "steins_reader_pages_served_total",
			Help: "Pages returned via the reader manifest",
		},
	)

	// Worker metrics
	taskDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "steins_worker_task_duration_seconds",
			Help:    "Duration of worker tasks",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60, 300},
		},
		[]string{"task_type", "status"},
	)

	queueDepth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "steins_queue_depth",
			Help: "Number of tasks waiting in each queue",
		},
		[]string{"queue"},
	)

	// Image processing metrics
	imagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "steins_images_processed_total",
			Help: "Total images processed",
		},
		[]string{"variant", "status"},
	)

	imageBytes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "steins_image_bytes_total",
			Help: "Bytes processed by the image worker",
		},
		[]string{"direction"}, // "in" / "out"
	)

	// Search metrics
	searchRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "steins_search_requests_total",
			Help: "Search requests by backend and outcome",
		},
		[]string{"backend", "status"},
	)
)
```

### Health Checks

Implement health check endpoints:

```go
type HealthCheck struct {
	Component   string
	Status      Status  // Healthy, Degraded, Unhealthy
	Message     string
	Latency     time.Duration
	LastChecked time.Time
}

type HealthChecker interface {
	Check(ctx context.Context) HealthCheck
}

type DatabaseHealthChecker struct {
	pool *pgxpool.Pool
}

func (h *DatabaseHealthChecker) Check(ctx context.Context) HealthCheck {
	start := time.Now()
	err := h.pool.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return HealthCheck{
			Component:   "database",
			Status:      Unhealthy,
			Message:     err.Error(),
			Latency:     latency,
			LastChecked: time.Now(),
		}
	}

	status := Healthy
	if latency > 100*time.Millisecond {
		status = Degraded
	}

	return HealthCheck{
		Component:   "database",
		Status:      status,
		Latency:     latency,
		LastChecked: time.Now(),
	}
}
```

Health check for:
- Database connectivity
- Redis connectivity
- Object storage (HEAD on a known key)
- Search backend
- Asynq queue connectivity
- Disk space
- Memory usage

### Alerting Rules

Define alerting conditions:

```yaml
groups:
  - name: steins
    interval: 30s
    rules:
      # High 5xx rate
      - alert: HighServerErrorRate
        expr: |
          sum by (route) (rate(steins_http_requests_total{status=~"5.."}[5m])) /
          sum by (route) (rate(steins_http_requests_total[5m])) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High 5xx rate on {{ $labels.route }}"

      # Slow reader
      - alert: ReaderLatencyP95High
        expr: |
          histogram_quantile(0.95,
            sum(rate(steins_http_request_duration_seconds_bucket{route="/api/v1/chapters/{id}/pages"}[5m]))
            by (le)
          ) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Reader p95 latency above 500ms"

      # Worker backlog
      - alert: WorkerQueueBacklog
        expr: steins_queue_depth > 5000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Queue {{ $labels.queue }} has {{ $value }} pending tasks"

      # Image worker failure
      - alert: ImageWorkerFailures
        expr: |
          sum(rate(steins_images_processed_total{status="failed"}[5m])) > 1
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Image worker producing failures"

      # Search backend down
      - alert: SearchBackendDown
        expr: |
          rate(steins_search_requests_total{backend="meilisearch",status="error"}[5m]) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Meilisearch backend appears unhealthy"
```

### Dashboards

Create Grafana dashboards for:

1. **Overview Dashboard**
   - Request rate / status mix
   - Reader pages served per second
   - Worker queue depths
   - Active sessions

2. **Reader Dashboard**
   - p50/p95/p99 latency for the page manifest
   - Cache hit ratio (Redis + CDN)
   - Image variant served distribution
   - Bandwidth out

3. **Worker Dashboard**
   - Tasks processed per task type
   - Average task duration
   - Retry / dead-letter counts
   - Image bytes in/out

4. **Storage Dashboard**
   - Postgres connections, query latency
   - Redis memory, eviction rate
   - Object storage error rate

5. **System Dashboard**
   - CPU/Memory per process
   - Goroutine count, GC pause
   - Disk space (logs, temp dirs)

## Incident Response

### On-Call Procedures

1. **Alert Response**
   - Check Grafana dashboards
   - Review recent logs (filter by `request_id` or `chapter_id`)
   - Identify affected components
   - Assess impact (reader unavailable? uploads stuck? search degraded?)

2. **Mitigation Steps**
   - For reader 5xx surge: roll back recent deploy, flush bad cache keys
   - For worker backlog: scale up worker concurrency
   - For storage failures: confirm credentials, switch to a backup bucket if configured
   - For database issues: confirm connection pool, fail over to read replica if primary is down

3. **Escalation**
   - P0 (Critical): Reader unavailable, data loss
   - P1 (High): Major feature broken (uploads stuck, login broken)
   - P2 (Medium): Search down, recommendations stale
   - P3 (Low): Minor issues, cosmetic regressions

### Debugging Tools

1. **Request Tracing**
   - Generate unique `request_id` per HTTP request (middleware)
   - Trace through services via `context.Context`
   - Include in logs and metrics labels

2. **Replay Capability**
   - Originals are preserved in object storage — re-run image processing without re-upload
   - Asynq UI lets operators retry / archive tasks

3. **Debug Endpoints** (admin-only)
   ```
   GET /debug/health        - extended health report
   GET /debug/queues        - Asynq queue stats
   POST /debug/chapters/:id/reprocess - re-enqueue chapter processing
   POST /debug/manga/:id/reindex      - re-enqueue search indexing
   ```

## Data Integrity

### Consistency Checks

1. **Referential Integrity**
   - Every `page` has a matching `chapter`
   - Every `chapter` has a matching `manga`
   - Every `library_entry` references a real user + manga

2. **Storage ↔ DB Reconciliation**
   - Daily job lists `originals/{chapter_id}/...` and compares to `pages` rows
   - Orphans (storage without DB) are flagged and queued for cleanup
   - Phantoms (DB without storage) raise an alert — never auto-delete

3. **Reader State**
   - `reading_progress.page_index` MUST be ≤ chapter.page_count
   - On chapter republish (`version` bump) — clamp progress to the new bounds

### Backup and Recovery

1. **Backup Strategy**
   - Postgres: Daily full backup, hourly WAL archive
   - Object storage: provider versioning + lifecycle rules
   - Redis: snapshots only for sessions/recommendations (regenerable cache OK to lose)
   - Config: version controlled in git

2. **Recovery Procedures**
   - Document restore procedures
   - Test recovery quarterly
   - Maintain runbooks in `docs/runbooks/`

3. **Data Retention**
   - Originals: indefinite
   - Derived variants: regeneratable, kept indefinitely but eligible for cleanup
   - Logs: 30 days
   - Metrics: 1 year (downsampled after 30 days)
   - Soft-deleted user data: 30 days, then hard-delete
