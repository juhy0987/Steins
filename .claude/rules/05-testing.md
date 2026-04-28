# Testing and Quality Assurance Rules

## Testing Philosophy

### Core Principles

1. **Test Coverage**
   - Minimum 70% code coverage for core packages
   - 90%+ coverage for critical paths (auth, reader endpoints, image processing)
   - 100% coverage for error handling paths

2. **Test Pyramid**
   ```
        E2E Tests (5%)
       ┌─────────────┐
       │Integration  │ (25%)
       ├─────────────┤
       │   Unit      │ (70%)
       └─────────────┘
   ```

3. **Test Isolation**
   - Tests MUST NOT depend on external services in unit-test mode
   - Use mocks/stubs for external dependencies in unit tests
   - Integration tests use testcontainers for real backends
   - Tests MUST be deterministic
   - Tests MUST be parallelizable

4. **Test Naming**
   ```go
   // Pattern: Test<Function>_<Scenario>_<Expected>
   func TestGetChapter_ValidID_ReturnsChapter(t *testing.T)
   func TestGetChapter_MissingID_ReturnsNotFound(t *testing.T)
   func TestProcessImage_LargeFile_ReturnsTimeoutError(t *testing.T)
   ```

## Unit Testing

### Testing Frameworks

1. **Standard Library**: `testing` package
2. **Assertions**: `github.com/stretchr/testify/assert` and `require`
3. **Mocking**: `github.com/stretchr/testify/mock`
4. **HTTP Mocking**: `httptest` package

### Unit Test Structure

```go
func TestChapterService_GetByID_Success(t *testing.T) {
	// Arrange
	repo := &mockChapterRepo{}
	repo.On("FindByID", mock.Anything, "ch-1").Return(&Chapter{ID: "ch-1", Title: "Test"}, nil)

	svc := NewChapterService(repo, nil, nil)

	// Act
	got, err := svc.GetByID(context.Background(), "ch-1")

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "ch-1", got.ID)
	assert.Equal(t, "Test", got.Title)
	repo.AssertExpectations(t)
}
```

### What to Test

1. **Handler Package**
   - Request decoding and validation
   - Status code mapping per error category
   - Successful happy path responses
   - Auth/role enforcement
   - Pagination/filter parsing

2. **Service Package**
   - Business rules (e.g., page index must not exceed chapter page_count)
   - Transaction boundaries (rollback on partial failure)
   - Idempotency
   - Permission checks

3. **Repository Package**
   - CRUD operations against a real DB (integration)
   - Query correctness
   - Transaction handling
   - Constraint violations

4. **Image Package**
   - Decode/encode round-trip
   - Resize aspect ratio preservation
   - EXIF stripping
   - Error handling on corrupt input

5. **Worker Package**
   - Payload decoding
   - Idempotent re-runs
   - Permanent vs retryable error classification
   - DLQ routing

### Test Data

1. **Fixtures**
   - Store sample images and JSON in `testdata/` directories
   - Use realistic examples (a few small JPEG/PNG/WebP samples)
   - Include edge cases (corrupt, oversized, transparent PNG, animated WebP)

   ```
   testdata/
   ├── images/
   │   ├── valid_color.jpg
   │   ├── large_4k.png
   │   ├── corrupt.jpg
   │   └── animated.webp
   ├── json/
   │   ├── create_chapter_valid.json
   │   └── create_chapter_invalid.json
   └── sql/
       ├── seed_manga.sql
       └── seed_users.sql
   ```

2. **Builders**
   - Use builder pattern for test objects
   ```go
   func NewTestManga() *Manga {
   	return &Manga{
   		ID:          "test-manga",
   		Slug:        "test-manga",
   		Title:       "Test Manga",
   		Status:      StatusOngoing,
   		Language:    "ko",
   		ReadingDir:  DirectionRTL,
   		PublishedAt: time.Now(),
   	}
   }

   func (m *Manga) WithTitle(title string) *Manga {
   	m.Title = title
   	return m
   }
   ```

### Table-Driven Tests

Use for testing multiple scenarios:

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
		{name: "plan to read", input: "plan-to-read", want: ShelfPlanToRead},
		{name: "invalid", input: "burning", wantErr: true},
		{name: "empty", input: "", wantErr: true},
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

## Integration Testing

### Database Integration Tests

```go
func TestChapterRepository_Insert(t *testing.T) {
	ctx := context.Background()

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:15-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": "test",
				"POSTGRES_DB":       "test",
			},
			WaitingFor: wait.ForLog("database system is ready to accept connections"),
		},
		Started: true,
	})
	require.NoError(t, err)
	defer pg.Terminate(ctx)

	pool := setupTestPool(t, pg)
	migrate(t, pool)

	repo := NewChapterRepository(pool)
	ch := NewTestChapter()

	err = repo.Insert(ctx, ch)
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, ch.ID, got.ID)
	assert.Equal(t, ch.Number, got.Number)
}
```

### Asynq Integration Tests

```go
func TestWorker_ProcessChapter_Idempotent(t *testing.T) {
	ctx := context.Background()
	redis := startRedis(t, ctx)
	defer redis.Terminate(ctx)

	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr(t, redis)})
	defer client.Close()

	// Enqueue twice — second run must be a no-op
	for i := 0; i < 2; i++ {
		_, err := client.Enqueue(asynq.NewTask("process_chapter",
			mustJSON(t, ProcessChapterPayload{ChapterID: "ch-1", Version: 1})))
		require.NoError(t, err)
	}

	processed := runWorker(t, ctx, redisAddr(t, redis))
	assert.Equal(t, 1, processed["ch-1"], "second run should detect existing derivatives")
}
```

### HTTP Integration Test

```go
func TestAPI_ReadChapterPages_ReturnsManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	srv := newTestServer(t)
	defer srv.Close()

	seedChapter(t, srv.DB, "ch-1", 5)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/chapters/ch-1/pages", nil)
	req.Header.Set("Authorization", "Bearer "+srv.UserToken)
	w := httptest.NewRecorder()

	srv.Router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Data struct {
			ChapterID string `json:"chapter_id"`
			Pages     []struct {
				Index int    `json:"index"`
				URL   string `json:"url"`
			} `json:"pages"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ch-1", body.Data.ChapterID)
	assert.Len(t, body.Data.Pages, 5)
}
```

### End-to-End Reader Test

```go
func TestE2E_UploadAndRead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	env := startEnvironment(t)  // postgres + redis + minio + worker
	defer env.Stop()

	// Upload a chapter via the API
	chapterID := uploadChapter(t, env, "manga-1", []string{
		"testdata/images/valid_color.jpg",
		"testdata/images/large_4k.png",
	})

	// Wait for the worker to flip status to ready
	waitForChapterReady(t, env, chapterID, 30*time.Second)

	// Read the manifest
	manifest := getPageManifest(t, env, chapterID)
	require.Len(t, manifest.Pages, 2)
	for _, p := range manifest.Pages {
		assert.Greater(t, p.Width, 0)
		assert.NotEmpty(t, p.URL)
	}
}
```

## Testing Strategies

### Mocking External Services

1. **Object Storage**
   ```go
   type ObjectStorage interface {
   	Put(ctx context.Context, key string, r io.Reader) error
   	Get(ctx context.Context, key string) (io.ReadCloser, error)
   	Head(ctx context.Context, key string) (ObjectInfo, error)
   	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
   }

   type MockObjectStorage struct{ mock.Mock }
   ```

2. **Database**
   - Use `pgxmock` or `sqlmock` for unit tests of repositories
   - Use testcontainers for integration tests
   - Avoid in-memory SQLite — schema differences hide real bugs

3. **Search Backend**
   ```go
   type SearchClient interface {
   	Index(ctx context.Context, name string, doc any) error
   	Delete(ctx context.Context, name string, id string) error
   	Query(ctx context.Context, name string, q SearchQuery) (*SearchResult, error)
   }
   ```

### Testing Async/Concurrent Code

```go
func TestImageProcessor_ConcurrentPages(t *testing.T) {
	proc := NewImageProcessor(ImageProcessorConfig{Concurrency: 4})
	chapter := NewTestChapter().WithPageCount(20)

	start := time.Now()
	err := proc.Process(context.Background(), chapter)
	elapsed := time.Since(start)

	require.NoError(t, err)
	// 20 pages at 4 concurrency should be < single-threaded baseline
	assert.Less(t, elapsed, 4*singlePageBaseline)
}
```

### Testing Rate Limiting

```go
func TestRateLimiter_EnforcesLimit(t *testing.T) {
	rdb := startRedis(t)
	limiter := NewRedisRateLimiter(rdb)

	for i := 0; i < 10; i++ {
		ok, _, err := limiter.Allow(context.Background(), "test-key", 10, time.Second)
		require.NoError(t, err)
		assert.True(t, ok)
	}

	ok, retryAfter, err := limiter.Allow(context.Background(), "test-key", 10, time.Second)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Greater(t, retryAfter, time.Duration(0))
}
```

## Benchmarking

### Performance Benchmarks

```go
func BenchmarkImageProcessor_Resize_PNG(b *testing.B) {
	src := loadTestImage(b, "testdata/images/large_4k.png")
	proc := NewImageProcessor(ImageProcessorConfig{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := proc.Resize(src, 1280)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPageManifest_Build(b *testing.B) {
	svc := newBenchmarkChapterService(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.GetPageManifest(context.Background(), "ch-1")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Test with different input sizes
func BenchmarkResize_Sizes(b *testing.B) {
	sizes := []int{640, 1280, 2048, 4096}
	src := loadTestImage(b, "testdata/images/large_4k.png")
	proc := NewImageProcessor(ImageProcessorConfig{})

	for _, size := range sizes {
		b.Run(fmt.Sprintf("width=%d", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = proc.Resize(src, size)
			}
		})
	}
}
```

### Memory Profiling

```bash
# Run with memory profiling
go test -bench=. -benchmem -memprofile=mem.prof

# Analyze
go tool pprof mem.prof
```

## Quality Gates

### Pre-Commit Checks

Create `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: local
    hooks:
      - id: go-test
        name: Go Tests
        entry: go test ./...
        language: system
        pass_filenames: false

      - id: go-vet
        name: Go Vet
        entry: go vet ./...
        language: system
        pass_filenames: false

      - id: golangci-lint
        name: golangci-lint
        entry: golangci-lint run
        language: system
        pass_filenames: false
```

### Linting

Use `golangci-lint` with strict config:

```yaml
# .golangci.yml
linters:
  enable:
    - gofmt
    - govet
    - errcheck
    - staticcheck
    - unused
    - gosimple
    - ineffassign
    - typecheck
    - gosec        # Security
    - gocritic     # Opinionated checks
    - gocyclo      # Complexity
    - dupl         # Duplicate code
    - misspell     # Spelling

linters-settings:
  gocyclo:
    min-complexity: 15
  govet:
    check-shadowing: true
  errcheck:
    check-blank: true

issues:
  max-same-issues: 0
  exclude-use-default: false
```

### Code Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View in browser
go tool cover -html=coverage.out

# Check minimum coverage
go test -cover ./... | grep "coverage:" | awk '{if ($2 < 70.0) exit 1}'
```

## Test Organization

### Directory Structure

모든 테스트 파일은 `test/` 디렉토리 아래에 위치하며, **서비스 아키텍처와 동일한 경로 구조**를 따른다.

**규칙:**
- `internal/<pkg-path>/` 의 테스트 → `test/internal/<pkg-path>/`
- `pkg/<pkg-path>/` 의 테스트 → `test/pkg/<pkg-path>/`
- 소스 파일과 같은 디렉토리에 테스트 파일을 두지 않는다.

```
test/                               # 모든 테스트 파일의 루트
├── internal/                       # internal/ 패키지 테스트
│   ├── api_handler/                # ← internal/api/handler/
│   │   ├── chapter_test.go
│   │   └── reader_test.go
│   ├── manga/                      # ← internal/manga/
│   │   └── service_test.go
│   ├── chapter/                    # ← internal/chapter/
│   │   └── service_test.go
│   ├── image/                      # ← internal/image/
│   │   └── processor_test.go
│   ├── worker/                     # ← internal/worker/
│   │   ├── process_chapter_test.go
│   │   └── index_manga_test.go
│   └── storage_postgres/           # ← internal/storage/postgres/
│       ├── chapter_repo_test.go
│       └── user_repo_test.go
└── pkg/                            # pkg/ 패키지 테스트
    ├── config/                     # ← pkg/config/
    │   └── config_test.go
    ├── httpx/                      # ← pkg/httpx/
    │   └── error_test.go
    └── logger/                     # ← pkg/logger/
        └── logger_test.go
```

**새 패키지에 테스트 추가 시:**
- 소스 경로 `internal/foo/bar/` → 테스트 경로 `test/internal/foo/bar/`
- 소스 경로 `pkg/foo/` → 테스트 경로 `test/pkg/foo/`
- 패키지 선언은 `package <name>_test` 형식 사용 (외부 테스트 패키지)

```go
// test/internal/chapter/service_test.go
package chapter_test

import "steins/internal/chapter"
```

### Test Suites

Group related tests:

```go
type ChapterHandlerTestSuite struct {
	suite.Suite
	srv *TestServer
}

func (s *ChapterHandlerTestSuite) SetupTest() {
	s.srv = newTestServer(s.T())
}

func (s *ChapterHandlerTestSuite) TearDownTest() {
	s.srv.Close()
}

func (s *ChapterHandlerTestSuite) TestGetPages_Success() {
	resp := s.srv.GET("/api/v1/chapters/ch-1/pages")
	s.Equal(http.StatusOK, resp.Code)
}

func TestChapterHandlerSuite(t *testing.T) {
	suite.Run(t, new(ChapterHandlerTestSuite))
}
```

## Continuous Integration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s

    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}

      - name: Run tests
        run: go test -race -coverprofile=coverage.out ./...

      - name: Check coverage
        run: |
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 70" | bc -l) )); then
            echo "Coverage $coverage% is below 70%"
            exit 1
          fi

      - name: Run linters
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
```

## Smoke Testing

### Production Smoke Tests

Run after deployment:

```go
func TestSmoke_HealthEndpoint(t *testing.T) {
	if os.Getenv("ENV") != "production" {
		t.Skip("Smoke tests only run in production")
	}

	resp, err := http.Get("https://api.steins.example/healthz")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSmoke_PublicCatalog(t *testing.T) {
	resp, err := http.Get("https://api.steins.example/api/v1/manga?page_size=1")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

## Load Testing

### Locust Configuration

```python
# locustfile.py
from locust import HttpUser, task, between

class ReaderUser(HttpUser):
	wait_time = between(1, 3)

	@task(5)
	def browse_catalog(self):
		self.client.get("/api/v1/manga?page_size=20")

	@task(10)
	def read_chapter(self):
		self.client.get("/api/v1/chapters/ch-1/pages")

	@task(2)
	def search(self):
		self.client.get("/api/v1/search?q=action")
```

Run load tests before major releases:

```bash
locust -f locustfile.py --host=https://api.steins.example
```

## Test Documentation

### Test Plans

Document test scenarios in `docs/testing/`:

```
docs/testing/
├── test-plan.md           # Overall test strategy
├── reader-tests.md        # Reader-specific scenarios
├── upload-tests.md        # Upload pipeline tests
└── edge-cases.md          # Known edge cases and handling
```

### Known Issues

Track known issues and limitations:

```markdown
# Known Test Limitations

1. **Animated WebP**
   - Issue: imaging library does not preserve animation frames
   - Workaround: reject animated WebP at upload validation
   - Test: `TestUploadValidator_AnimatedWebP_Rejected`

2. **Object Storage Eventual Consistency**
   - Issue: PUT-then-HEAD can return 404 on some providers
   - Approach: integration tests retry HEAD with backoff
   - Test: `TestObjectStorage_PutHead_Eventually`
```
