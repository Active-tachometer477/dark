package dark

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// MiddlewareFunc is a standard Go HTTP middleware.
type MiddlewareFunc func(http.Handler) http.Handler

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (sr *statusRecorder) WriteHeader(code int) {
	sr.status = code
	sr.ResponseWriter.WriteHeader(code)
}

// Logger returns a middleware that logs each request.
func Logger() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start),
			)
		})
	}
}

// Recover returns a middleware that recovers from panics and returns a 500 response.
func Recover() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					slog.Error("panic recovered", "error", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RecoverWithErrorPage returns a middleware that recovers from panics and renders
// the configured error page. Use this instead of Recover() to get custom error pages.
func (app *App) RecoverWithErrorPage() MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("panic recovered", "error", rec)
					err, ok := rec.(error)
					if !ok {
						err = fmt.Errorf("panic: %v", rec)
					}
					app.renderError(w, r, err)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
