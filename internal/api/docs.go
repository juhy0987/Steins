package api

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.yaml
var openapiYAML []byte

// docsHTML is the Swagger UI entry page. It loads swagger-ui-dist from a
// pinned-major CDN so the binary stays free of frontend assets — Swagger UI
// is a developer tool, not a production reader-facing page.
//
// docsHTML 은 Swagger UI 진입 페이지입니다. swagger-ui-dist 를 고정 메이저
// 버전의 CDN 에서 로드하므로 바이너리에 프론트엔드 에셋이 포함되지 않습니다.
const docsHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<title>Steins API Docs</title>
	<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
	<style>body { margin: 0; }</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js" crossorigin></script>
	<script>
		window.addEventListener('load', function () {
			window.ui = SwaggerUIBundle({
				url: '/api/v1/openapi.yaml',
				dom_id: '#swagger-ui',
				deepLinking: true,
				presets: [SwaggerUIBundle.presets.apis],
				layout: 'BaseLayout',
			});
		});
	</script>
</body>
</html>
`

// serveOpenAPISpec returns the embedded OpenAPI 3.0 specification.
func serveOpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	// 스펙은 바이너리에 임베드되어 메모리에서 서빙되므로 캐시 없이도 비용이 거의 없다.
	// 개발 중 spec 변경 후 서버 재기동 시 브라우저가 stale 캐시를 보지 않도록 no-cache.
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(openapiYAML)
}

// serveSwaggerUI returns the Swagger UI HTML entry page.
func serveSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// HTML 도 임베드된 상수이므로 캐시 비용이 없고, UI 설정 변경 시 즉시 반영되도록 no-cache.
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(docsHTML))
}
