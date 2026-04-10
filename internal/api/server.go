// Package api hosts the HTTP server, router, and top-level request handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server wraps a net/http server configured with the sonarr2 router.
type Server struct {
	log     *slog.Logger
	httpsrv *http.Server
}

// New builds a Server bound to cfg.BindAddress:cfg.Port.
func New(cfg config.HTTPConfig, log *slog.Logger) *Server {
	return &Server{
		log: log,
		httpsrv: &http.Server{
			Addr:              net.JoinHostPort(cfg.BindAddress, strconv.Itoa(cfg.Port)),
			Handler:           Handler(log),
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

// Handler builds the chi router without wrapping it in a full server. Useful
// for tests that need to drive the handler directly.
func Handler(log *slog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)

	r.Get("/ping", pingHandler)
	r.Get("/api/v3/system/status", statusHandler)

	return r
}

func pingHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	info := buildinfo.Get()
	resp := map[string]any{
		"appName":   "sonarr2",
		"version":   info.Version,
		"buildTime": info.Date,
		"commit":    info.Commit,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
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
