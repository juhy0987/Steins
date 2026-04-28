// Package middleware provides HTTP middlewares — request id, structured
// logging, recover, and CORS — used by the API server.
//
// middleware 패키지는 API 서버가 사용하는 request id, structured logging,
// recover, CORS 미들웨어를 제공합니다.
package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	chi "github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"steins/pkg/logger"
)

// RequestID is re-exported from chi so callers don't have to know about chi.
var RequestID = chimw.RequestID

// statusRecorder wraps ResponseWriter to capture the status code for logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
	size   int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.size += n
	return n, err
}

// Logging emits a structured log entry per HTTP request.
//
// Logging은 HTTP 요청마다 structured 로그를 출력합니다.
func Logging(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}

			// Inject the request-scoped logger so handlers can pull it via context.
			reqID, _ := r.Context().Value(chimw.RequestIDKey).(string)
			scoped := log.WithFields(map[string]interface{}{
				"request_id": reqID,
				"method":     r.Method,
				"route":      routePattern(r),
			})
			ctx := scoped.ToContext(r.Context())

			defer func() {
				ev := scoped.Info().
					Int("status_code", rec.status).
					Int64("duration_ms", time.Since(start).Milliseconds())
				if rec.status >= 500 {
					ev = scoped.Error().
						Int("status_code", rec.status).
						Int64("duration_ms", time.Since(start).Milliseconds())
				}
				ev.Msg("request completed")
			}()

			next.ServeHTTP(rec, r.WithContext(ctx))
		})
	}
}

// Recover catches panics, logs them with stack, and returns 500.
//
// Recover는 panic을 잡아 stack trace와 함께 로그를 남기고 500을 반환합니다.
func Recover(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error().
						Str("stack", string(debug.Stack())).
						Msg("panic recovered")
					http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"unexpected error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS allows the React dev frontend (and any configured origins) to talk
// to the API. v0 keeps it permissive for development; production should
// narrow Allow-Origin via configuration.
//
// CORS는 React dev frontend(및 설정된 origin)가 API와 통신할 수 있게 합니다.
// v0에서는 개발 편의를 위해 허용 범위를 넓게 두며, 프로덕션에서는 설정으로
// Allow-Origin을 제한해야 합니다.
func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	allowAll := len(allowedOrigins) == 0

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				} else if _, ok := allowed[origin]; ok {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "Digest, ETag, Content-Length")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	return r.URL.Path
}
