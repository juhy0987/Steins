# Steins

> Go 기반 만화 뷰어 API 서버 (v0)

React 프론트엔드와 짝을 이루는 Go API 서버입니다. v0 범위는 **로그인 없이 만화를 조회·읽기**위한 최소 API와, 전송 중 변조를 검출할 수 있도록 각 페이지 이미지의 **SHA-256 체크섬**을 매니페스트와 응답 헤더로 함께 제공하는 무결성 검증 기능입니다.

## 기능

- 만화 카탈로그 / 시리즈 상세 / 챕터 목록 조회
- 챕터 페이지 매니페스트 (페이지별 SHA-256 체크섬 포함)
- 이미지 바이트 스트리밍 + `Digest: sha-256=<base64>` 응답 헤더 (RFC 9530)
- 파일시스템 기반 카탈로그 (`data/manga/...`) — v0 단순 구현, 후속 이슈에서 PostgreSQL 으로 교체

## 요구사항

- Go 1.22+
- (선택) `golangci-lint`

## 빠른 시작

```bash
# 1) 빌드
make build            # bin/api, bin/seed

# 2) 샘플 데이터 시드 — 더미 PNG 페이지 생성
make seed-data        # data/manga/<slug>/...

# 3) 서버 실행 (기본 :8080, 사용 중이면 ADDR 지정)
make run
# 또는
./bin/api -addr :18080 -data ./data/manga -pretty
```

## API 엔드포인트

| Method | Path | Description |
|--------|------|-------------|
| GET | `/healthz` | liveness 체크 |
| GET | `/api/v1/manga` | 만화 목록 |
| GET | `/api/v1/manga/{slug}` | 만화 상세 |
| GET | `/api/v1/manga/{slug}/chapters` | 챕터 목록 |
| GET | `/api/v1/chapters/{id}/pages` | **체크섬 포함** 페이지 매니페스트 |
| GET | `/api/v1/chapters/{id}/pages/{index}/image` | 이미지 바이트 (`Digest` 헤더 포함) |

## 무결성 검증 흐름

1. 클라이언트가 `GET /api/v1/chapters/{id}/pages` 호출
2. 응답의 각 페이지에는 `checksum: "sha-256:<base64>"` 가 포함됨
3. 클라이언트가 `pages[].url` 로 이미지 바이트를 받음
4. 다운로드 바이트의 SHA-256 을 계산 → 매니페스트의 체크섬 또는 응답 헤더의 `Digest: sha-256=<base64>` 와 비교
5. 불일치 시 변조 가능성 → 페이지 표시 거부 또는 재요청

### 예시

```bash
# 매니페스트 조회
curl -s http://localhost:18080/api/v1/chapters/steins-sample__0001/pages

# 이미지 응답 헤더 확인
curl -sD - -o /dev/null \
  http://localhost:18080/api/v1/chapters/steins-sample__0001/pages/1/image

# 본문 SHA-256 검증 (base64)
curl -s http://localhost:18080/api/v1/chapters/steins-sample__0001/pages/1/image \
  | sha256sum | awk '{print $1}' | xxd -r -p | base64
```

## 프로젝트 구조

```
cmd/api          # API 서버 entrypoint
cmd/seed         # 샘플 데이터 생성기
internal/api     # 라우터, 핸들러, 미들웨어
internal/manga   # manga 도메인 (model/repo/service)
internal/chapter # chapter/page 도메인
internal/integrity # SHA-256 체크섬 계산기 (캐시)
internal/storage/fs # 파일시스템 기반 repository
internal/apperr  # 구조화 에러 타입
pkg/httpx        # JSON / 에러 응답 헬퍼
pkg/logger       # zerolog wrapper
test/internal    # 외부 테스트 패키지
data/manga       # 샘플 카탈로그 (gitignore: 이미지 바이트는 시드로 재생성)
```

## 테스트

```bash
make test       # go test -race ./...
```

## 후속 작업

- PostgreSQL 기반 repository로 교체 (스토리지 인터페이스만 변경)
- React 프론트 개발 + CORS 설정 정밀화
- 인증/회원/라이브러리/검색 (이슈로 분리)
