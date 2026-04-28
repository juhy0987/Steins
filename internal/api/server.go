// Package api wires the HTTP router for the Steins API.
//
// api 패키지는 Steins API의 HTTP 라우터를 구성합니다.
package api

import (
	"net/http"

	chi "github.com/go-chi/chi/v5"

	"steins/internal/api/handler"
	"steins/internal/api/middleware"
	"steins/pkg/logger"
)

// Config controls server-level wiring options.
type Config struct {
	AllowedOrigins []string
}

// NewRouter returns a fully wired chi router.
func NewRouter(cfg Config, log *logger.Logger, h *handler.Handlers) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(log))
	r.Use(middleware.Recover(log))
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.Get("/healthz", h.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/manga", h.ListManga)
		r.Get("/manga/{slug}", h.GetManga)
		r.Get("/manga/{slug}/chapters", h.ListChapters)
		r.Get("/chapters/{id}/pages", h.GetPageManifest)
		r.Get("/chapters/{id}/pages/{index}/image", h.ServePageImage)
	})

	return r
}
