// Package api hosts the HTTP server, router, and top-level request handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	v3 "github.com/ajthom90/sonarr2/internal/api/v3"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/realtime"
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

// Deps holds the optional domain stores used by v3 handlers. Fields may be
// nil; if nil the corresponding v3 routes are not mounted. Task 7 will
// consolidate this into a proper v3.Dependencies struct.
type Deps struct {
	Pool            PoolPinger
	HostConfig      hostconfig.Store
	Series          library.SeriesStore
	Seasons         library.SeasonsStore
	Stats           library.SeriesStatsStore
	Episodes        library.EpisodesStore
	EpisodeFiles    library.EpisodeFilesStore
	QualityProfiles profiles.QualityProfileStore
	QualityDefs     profiles.QualityDefinitionStore
	CustomFormats   customformats.Store
	Commands        commands.Queue
	History         history.Store
	IndexerStore    indexer.InstanceStore
	IndexerRegistry *indexer.Registry
	DCStore         downloadclient.InstanceStore
	DCRegistry      *downloadclient.Registry
	Broker          *realtime.Broker
	Log             *slog.Logger
}

// Server wraps a net/http server configured with the sonarr2 router.
type Server struct {
	log     *slog.Logger
	httpsrv *http.Server
}

// New builds a Server bound to addr. Prefer NewWithDeps when domain stores
// are available.
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

// NewWithDeps builds a Server with full deps, including v3 route mounting
// and API key auth.
func NewWithDeps(addr string, log *slog.Logger, deps Deps) *Server {
	return &Server{
		log: log,
		httpsrv: &http.Server{
			Addr:              addr,
			Handler:           HandlerWithDeps(log, deps),
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

// HandlerWithDeps builds the chi router with full deps: API key auth and v3
// routes. Use this in production; use HandlerWithPool / Handler for tests
// that only need the status or ping endpoints.
func HandlerWithDeps(log *slog.Logger, deps Deps) http.Handler {
	if deps.Log != nil {
		log = deps.Log
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestLogger(log))
	r.Use(middleware.Recoverer)

	// /ping is a liveness check — no auth required.
	r.Get("/ping", pingHandler)

	// SignalR routes — outside the auth group because SignalR clients negotiate
	// before presenting credentials (Sonarr's convention). Mounted only when a
	// broker is wired in.
	if deps.Broker != nil {
		r.Post("/signalr/messages/negotiate", deps.Broker.SignalRNegotiate)
		r.Get("/signalr/messages", deps.Broker.SignalRConnect)
	}

	// All v3 routes are gated behind API key auth when HostConfig is set.
	if deps.HostConfig == nil {
		// Fall back to the unauthenticated stub for test/minimal configurations.
		r.Get("/api/v3/system/status", statusHandlerWithPool(deps.Pool))
		return r
	}

	r.Group(func(r chi.Router) {
		r.Use(apiKeyAuth(deps.HostConfig))

		// system/status: uses the richer v3 handler with full Sonarr field parity.
		ssh := v3.NewSystemStatusHandler(deps.Pool)
		v3.MountSystemStatus(r, ssh)

		// Task 3 — series.
		if deps.Series != nil && deps.Seasons != nil && deps.Stats != nil {
			sh := v3.NewSeriesHandler(deps.Series, deps.Seasons, deps.Stats, log)
			v3.MountSeries(r, sh)
		}

		// Task 4 — episode + episodefile.
		if deps.Episodes != nil {
			eh := v3.NewEpisodeHandler(deps.Episodes, log)
			v3.MountEpisode(r, eh)
		}
		if deps.EpisodeFiles != nil && deps.Series != nil {
			efh := v3.NewEpisodeFileHandler(deps.EpisodeFiles, deps.Series, log)
			v3.MountEpisodeFile(r, efh)
		}

		// Task 5 — quality, customformats, command, history, calendar.
		if deps.QualityProfiles != nil && deps.QualityDefs != nil {
			qph := v3.NewQualityProfileHandler(deps.QualityProfiles, deps.QualityDefs, log)
			v3.MountQualityProfile(r, qph)
		}
		if deps.QualityDefs != nil {
			qdh := v3.NewQualityDefinitionHandler(deps.QualityDefs, log)
			v3.MountQualityDefinition(r, qdh)
		}
		if deps.CustomFormats != nil {
			cfh := v3.NewCustomFormatHandler(deps.CustomFormats, log)
			v3.MountCustomFormat(r, cfh)
		}
		if deps.Commands != nil {
			ch := v3.NewCommandHandler(deps.Commands, log)
			v3.MountCommand(r, ch)
		}
		if deps.History != nil {
			hh := v3.NewHistoryHandler(deps.History, log)
			v3.MountHistory(r, hh)
		}
		if deps.Episodes != nil {
			cal := v3.NewCalendarHandler(deps.Episodes, log)
			v3.MountCalendar(r, cal)
		}

		// Task 6 — providers + utility.
		if deps.IndexerStore != nil && deps.IndexerRegistry != nil {
			ih := v3.NewIndexerHandler(deps.IndexerStore, deps.IndexerRegistry, log)
			v3.MountIndexer(r, ih)
		}
		if deps.DCStore != nil && deps.DCRegistry != nil {
			dch := v3.NewDownloadClientHandler(deps.DCStore, deps.DCRegistry, log)
			v3.MountDownloadClient(r, dch)
		}
		if deps.Series != nil {
			rfh := v3.NewRootFolderHandler(deps.Series, log)
			v3.MountRootFolder(r, rfh)
		}
		v3.MountTag(r)
		v3.MountHealth(r)
		v3.MountParse(r)
		if deps.Episodes != nil {
			wh := v3.NewWantedHandler(deps.Episodes, log)
			v3.MountWanted(r, wh)
		}

		// SSE transport — behind auth so only authenticated clients connect.
		if deps.Broker != nil {
			r.Get("/api/v6/stream", deps.Broker.SSEHandler)
		}
	})

	return r
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
