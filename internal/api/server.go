// Package api hosts the HTTP server, router, and top-level request handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// PoolPinger is the minimum interface the status handler needs to report
// database connectivity. The api package intentionally does not import
// internal/db to keep this layer free of database-package coupling — the
// app composition root adapts a db.Pool to this interface.
type PoolPinger interface {
	Dialect() string
	Ping(ctx context.Context) error
}

// Server wraps a net/http server configured with the sonarr2 router.
type Server struct {
	log     *slog.Logger
	httpsrv *http.Server
}

// New builds a Server bound to addr.
func New(addr string, log *slog.Logger, pool PoolPinger) *Server {
	return &Server{
		log: log,
		httpsrv: &http.Server{
			Addr:              addr,
			Handler:           HandlerWithPool(log, pool),
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Start blocks until the server stops or errors. ErrServerClosed from a clean
// Shutdown is not returned as an error.
func (s *Server) Start() error {
	s.log.Info("http server listening", slog.String("addr", s.httpsrv.Addr))
	if err := s.httpsrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server within the context deadline.
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("http server shutting down")
	return s.httpsrv.Shutdown(ctx)
}

// HandlerWithPool builds the chi router with a database pool reference so
// the /api/v3/system/status handler can report db connectivity. Most code
// should use this instead of Handler.
func HandlerWithPool(log *slog.Logger, pool PoolPinger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/ping", pingHandler)
	r.Get("/api/v3/system/status", statusHandlerWithPool(pool))

	return r
}

// Handler builds the chi router without wrapping it in a full server.
// Convenience for tests that don't need a pool; the status handler tolerates
// a nil pool and reports database connected=false.
func Handler(log *slog.Logger) http.Handler {
	return HandlerWithPool(log, nil)
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func statusHandlerWithPool(pool PoolPinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info := buildinfo.Get()
		resp := map[string]any{
			"appName":   "sonarr2",
			"version":   info.Version,
			"buildTime": info.Date,
			"commit":    info.Commit,
		}

		if pool != nil {
			connected := pool.Ping(r.Context()) == nil
			resp["database"] = map[string]any{
				"dialect":   pool.Dialect(),
				"connected": connected,
			}
		} else {
			resp["database"] = map[string]any{
				"dialect":   "",
				"connected": false,
			}
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			defer func() {
				log.Info("http request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.Status()),
					slog.Duration("dur", time.Since(start)),
					slog.String("request_id", middleware.GetReqID(r.Context())),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}
