// Package api hosts the HTTP server, router, and top-level request handlers.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	v3 "github.com/ajthom90/sonarr2/internal/api/v3"
	v6 "github.com/ajthom90/sonarr2/internal/api/v6"
	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/backup"
	"github.com/ajthom90/sonarr2/internal/blocklist"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/delayprofile"
	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/providers/notification"
	"github.com/ajthom90/sonarr2/internal/realtime"
	"github.com/ajthom90/sonarr2/internal/releaseprofile"
	"github.com/ajthom90/sonarr2/internal/remotepathmapping"
	"github.com/ajthom90/sonarr2/internal/tags"
	"github.com/ajthom90/sonarr2/web"
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
	Pool                 PoolPinger
	HostConfig           hostconfig.Store
	Series               library.SeriesStore
	Seasons              library.SeasonsStore
	Stats                library.SeriesStatsStore
	Episodes             library.EpisodesStore
	EpisodeFiles         library.EpisodeFilesStore
	QualityProfiles      profiles.QualityProfileStore
	QualityDefs          profiles.QualityDefinitionStore
	CustomFormats        customformats.Store
	Tags                 tags.Store
	Blocklist            blocklist.Store
	RemotePathMappings   remotepathmapping.Store
	ReleaseProfiles      releaseprofile.Store
	DelayProfiles        delayprofile.Store
	Commands             commands.Queue
	History              history.Store
	IndexerStore         indexer.InstanceStore
	IndexerRegistry      *indexer.Registry
	DCStore              downloadclient.InstanceStore
	DCRegistry           *downloadclient.Registry
	NotificationStore    notification.InstanceStore
	NotificationRegistry *notification.Registry
	MetadataSource       metadatasource.MetadataSource
	UserStore            auth.UserStore
	SessionStore         auth.SessionStore
	HealthChecker        *health.Checker
	Broker               *realtime.Broker
	BackupService        *backup.Service
	OnTvdbKeyChanged     func(string)
	Log                  *slog.Logger
	URLBase              string
	RateLimit            float64
	RateBurst            int
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
	r.Use(securityHeaders)
	r.Use(corsMiddleware)

	if deps.RateLimit > 0 {
		rl := newIPRateLimiter(deps.RateLimit, deps.RateBurst)
		r.Use(rl.Middleware)
	}

	// mountRoutes registers all application routes on the provided router.
	// When URLBase is set, this is called inside r.Route(urlBase, ...) so all
	// routes are nested under the prefix; otherwise it is called directly on r.
	mountRoutes := func(r chi.Router) {
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
			return
		}

		// Auth endpoints — outside auth group (must be accessible without auth).
		if deps.UserStore != nil && deps.SessionStore != nil {
			v3.MountAuth(r, deps.UserStore, deps.SessionStore, deps.HostConfig)
		}

		r.Group(func(r chi.Router) {
			if deps.SessionStore != nil {
				r.Use(combinedAuth(deps.HostConfig, deps.SessionStore))
			} else {
				r.Use(apiKeyAuth(deps.HostConfig))
			}

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
				if deps.Series != nil {
					feed := v3.NewCalendarFeedHandler(deps.Episodes, deps.Series, log)
					v3.MountCalendarFeed(r, feed)
				}
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
			if deps.NotificationStore != nil && deps.NotificationRegistry != nil {
				nh := v3.NewNotificationHandler(deps.NotificationStore, deps.NotificationRegistry, log)
				v3.MountNotification(r, nh)
			}
			if deps.MetadataSource != nil {
				v3.MountSeriesLookup(r, deps.MetadataSource)
			}
			if deps.Series != nil {
				rfh := v3.NewRootFolderHandler(deps.Series, log)
				v3.MountRootFolder(r, rfh)
			}
			if deps.Tags != nil {
				th := v3.NewTagHandler(deps.Tags, log)
				v3.MountTag(r, th)
			} else {
				v3.MountTag(r, nil)
			}
			if deps.Blocklist != nil {
				bh := v3.NewBlocklistHandler(deps.Blocklist, log)
				v3.MountBlocklist(r, bh)
			}
			if deps.RemotePathMappings != nil {
				rpmh := v3.NewRemotePathMappingHandler(deps.RemotePathMappings, log)
				v3.MountRemotePathMapping(r, rpmh)
			}
			if deps.ReleaseProfiles != nil {
				rph := v3.NewReleaseProfileHandler(deps.ReleaseProfiles, log)
				v3.MountReleaseProfile(r, rph)
			}
			if deps.DelayProfiles != nil {
				dph := v3.NewDelayProfileHandler(deps.DelayProfiles, log)
				v3.MountDelayProfile(r, dph)
			}
			v3.MountHealth(r, deps.HealthChecker)
			v3.MountParse(r)
			if deps.Episodes != nil {
				wh := v3.NewWantedHandler(deps.Episodes, log)
				v3.MountWanted(r, wh)
			}
			if deps.BackupService != nil {
				v3.MountBackup(r, deps.BackupService)
			}

			// General settings.
			if deps.HostConfig != nil {
				v3.MountSettings(r, deps.HostConfig, deps.OnTvdbKeyChanged)
			}

			// SSE transport — behind auth so only authenticated clients connect.
			if deps.Broker != nil {
				r.Get("/api/v6/stream", deps.Broker.SSEHandler)
			}
		})

		// v6 routes — mounted separately under /api/v6 with their own auth group.
		v6.Mount(r, v6.Deps{
			Pool:                 deps.Pool,
			HostConfig:           deps.HostConfig,
			SessionStore:         deps.SessionStore,
			Series:               deps.Series,
			Seasons:              deps.Seasons,
			Stats:                deps.Stats,
			Episodes:             deps.Episodes,
			EpisodeFiles:         deps.EpisodeFiles,
			QualityProfiles:      deps.QualityProfiles,
			QualityDefs:          deps.QualityDefs,
			CustomFormats:        deps.CustomFormats,
			Commands:             deps.Commands,
			History:              deps.History,
			IndexerStore:         deps.IndexerStore,
			IndexerRegistry:      deps.IndexerRegistry,
			DCStore:              deps.DCStore,
			DCRegistry:           deps.DCRegistry,
			NotificationStore:    deps.NotificationStore,
			NotificationRegistry: deps.NotificationRegistry,
			MetadataSource:       deps.MetadataSource,
			HealthChecker:        deps.HealthChecker,
			BackupService:        deps.BackupService,
			OnTvdbKeyChanged:     deps.OnTvdbKeyChanged,
			Log:                  deps.Log,
		})

		// Frontend SPA — served from the embedded web/dist directory.
		// All paths not matched above fall through to the SPA handler, which
		// returns index.html for unknown routes (client-side routing).
		r.Handle("/*", spaHandler())
	}

	urlBase := strings.TrimRight(deps.URLBase, "/")
	if urlBase != "" {
		r.Route(urlBase, mountRoutes)
	} else {
		mountRoutes(r)
	}

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

// spaHandler returns an http.Handler that serves embedded frontend files.
// For paths that exist in the embedded FS it serves the file directly.
// For all other paths it serves index.html, supporting client-side routing.
func spaHandler() http.Handler {
	sub, err := web.DistFS()
	if err != nil {
		// Fallback: web/dist wasn't embedded (should not happen with placeholder).
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not available", http.StatusServiceUnavailable)
		})
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Strip the leading "/" so we can probe the embedded FS.
		stripped := strings.TrimPrefix(r.URL.Path, "/")
		if stripped == "" {
			stripped = "index.html"
		}
		f, openErr := sub.Open(stripped)
		if openErr == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		// SPA fallback: unknown paths serve index.html.
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
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
