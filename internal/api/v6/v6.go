// Package v6 implements the sonarr2 v6 REST API surface.
// v6 uses cursor pagination, RFC 9457 problem-detail error envelopes,
// and clean JSON shapes without legacy Sonarr cruft.
package v6

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/backup"
	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// PoolPinger is the minimum interface needed for system/status DB reporting.
type PoolPinger interface {
	Dialect() string
	Ping(ctx context.Context) error
}

// Deps holds the dependencies needed by v6 handlers. It mirrors api.Deps but
// is defined here to avoid an import cycle between api and api/v6.
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
	Commands             commands.Queue
	History              history.Store
	IndexerStore         indexer.InstanceStore
	IndexerRegistry      *indexer.Registry
	DCStore              downloadclient.InstanceStore
	DCRegistry           *downloadclient.Registry
	NotificationStore    notification.InstanceStore
	NotificationRegistry *notification.Registry
	SessionStore         auth.SessionStore
	HealthChecker        *health.Checker
	BackupService        *backup.Service
	Log                  *slog.Logger
}

// combinedAuth accepts either a valid API key or a valid session cookie.
func combinedAuth(hcStore hostconfig.Store, sessionStore auth.SessionStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check API key.
			key := r.Header.Get("X-Api-Key")
			if key == "" {
				key = r.URL.Query().Get("apikey")
			}
			if key != "" {
				hc, err := hcStore.Get(r.Context())
				if err == nil && hc.APIKey == key {
					next.ServeHTTP(w, r)
					return
				}
			}

			// 2. Check session cookie.
			if sessionStore != nil {
				cookie, err := r.Cookie("sonarr2_session")
				if err == nil && cookie.Value != "" {
					session, err := sessionStore.GetByToken(r.Context(), cookie.Value)
					if err == nil && time.Now().Before(session.ExpiresAt) {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"Unauthorized"}`))
		})
	}
}

// Mount registers all v6 routes on the given chi router under /api/v6.
func Mount(r chi.Router, deps Deps) {
	r.Route("/api/v6", func(r chi.Router) {
		if deps.HostConfig != nil {
			r.Use(combinedAuth(deps.HostConfig, deps.SessionStore))
		}

		// series
		if deps.Series != nil && deps.Seasons != nil && deps.Stats != nil {
			sh := newSeriesHandler(deps.Series, deps.Seasons, deps.Stats, deps.Log)
			mountSeries(r, sh)
		}

		// episode
		if deps.Episodes != nil {
			eh := newEpisodeHandler(deps.Episodes, deps.Log)
			mountEpisode(r, eh)
		}

		// episodefile
		if deps.EpisodeFiles != nil && deps.Series != nil {
			efh := newEpisodeFileHandler(deps.EpisodeFiles, deps.Series, deps.Log)
			mountEpisodeFile(r, efh)
		}

		// qualityprofile
		if deps.QualityProfiles != nil && deps.QualityDefs != nil {
			qph := newQualityProfileHandler(deps.QualityProfiles, deps.QualityDefs, deps.Log)
			mountQualityProfile(r, qph)
		}

		// customformat
		if deps.CustomFormats != nil {
			cfh := newCustomFormatHandler(deps.CustomFormats, deps.Log)
			mountCustomFormat(r, cfh)
		}

		// command
		if deps.Commands != nil {
			ch := newCommandHandler(deps.Commands, deps.Log)
			mountCommand(r, ch)
		}

		// history
		if deps.History != nil {
			hh := newHistoryHandler(deps.History, deps.Log)
			mountHistory(r, hh)
		}

		// calendar
		if deps.Episodes != nil {
			cal := newCalendarHandler(deps.Episodes, deps.Log)
			mountCalendar(r, cal)
		}

		// indexer
		if deps.IndexerStore != nil && deps.IndexerRegistry != nil {
			ih := newIndexerHandler(deps.IndexerStore, deps.IndexerRegistry, deps.Log)
			mountIndexer(r, ih)
		}

		// downloadclient
		if deps.DCStore != nil && deps.DCRegistry != nil {
			dch := newDownloadClientHandler(deps.DCStore, deps.DCRegistry, deps.Log)
			mountDownloadClient(r, dch)
		}

		// notification
		if deps.NotificationStore != nil && deps.NotificationRegistry != nil {
			nh := newNotificationHandler(deps.NotificationStore, deps.NotificationRegistry, deps.Log)
			mountNotification(r, nh)
		}

		// system/status
		mountSystemStatus(r, deps.Pool)

		// health
		if deps.HealthChecker != nil {
			mountHealth(r, deps.HealthChecker)
		}

		// backup
		if deps.BackupService != nil {
			mountBackup(r, deps.BackupService)
		}

		// parse
		mountParse(r)
	})
}

// parseIDFromRequest extracts and parses the {id} URL parameter as int64.
func parseIDFromRequest(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}
