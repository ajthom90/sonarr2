// Package app is the composition root for sonarr2 — it wires the logger,
// HTTP server, and graceful shutdown together.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"path/filepath"

	"github.com/ajthom90/sonarr2/internal/api"
	"github.com/ajthom90/sonarr2/internal/auth"
	"github.com/ajthom90/sonarr2/internal/backup"
	"github.com/ajthom90/sonarr2/internal/blocklist"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/commands/handlers"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/delayprofile"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/fswatcher"
	"github.com/ajthom90/sonarr2/internal/grab"
	"github.com/ajthom90/sonarr2/internal/health"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/housekeeping"
	"github.com/ajthom90/sonarr2/internal/importer"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/logging"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/aria2"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/blackhole"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/blackholetorrent"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/blackholeusenet"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/deluge"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/dstation"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/flood"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/freebox"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/hadouken"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/nzbget"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/nzbvortex"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/pneumatic"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/qbittorrent"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/rqbit"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/rtorrent"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/sabnzbd"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/transmission"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/tribler"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/utorrent"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/vuze"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/broadcasthenet"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/iptorrents"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/newznab"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/nyaa"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/torrentrss"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/torznab"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/cached"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/tvdb"
	"github.com/ajthom90/sonarr2/internal/providers/notification"
	"github.com/ajthom90/sonarr2/internal/providers/notification/apprise"
	"github.com/ajthom90/sonarr2/internal/providers/notification/customscript"
	"github.com/ajthom90/sonarr2/internal/providers/notification/discord"
	notifyemail "github.com/ajthom90/sonarr2/internal/providers/notification/email"
	"github.com/ajthom90/sonarr2/internal/providers/notification/emby"
	"github.com/ajthom90/sonarr2/internal/providers/notification/gotify"
	notifyjoin "github.com/ajthom90/sonarr2/internal/providers/notification/join"
	"github.com/ajthom90/sonarr2/internal/providers/notification/kodi"
	"github.com/ajthom90/sonarr2/internal/providers/notification/mailgun"
	"github.com/ajthom90/sonarr2/internal/providers/notification/notifiarr"
	"github.com/ajthom90/sonarr2/internal/providers/notification/ntfy"
	"github.com/ajthom90/sonarr2/internal/providers/notification/plex"
	"github.com/ajthom90/sonarr2/internal/providers/notification/prowl"
	"github.com/ajthom90/sonarr2/internal/providers/notification/pushbullet"
	"github.com/ajthom90/sonarr2/internal/providers/notification/pushcut"
	"github.com/ajthom90/sonarr2/internal/providers/notification/pushover"
	"github.com/ajthom90/sonarr2/internal/providers/notification/sendgrid"
	notifysignal "github.com/ajthom90/sonarr2/internal/providers/notification/signal"
	"github.com/ajthom90/sonarr2/internal/providers/notification/simplepush"
	"github.com/ajthom90/sonarr2/internal/providers/notification/slack"
	"github.com/ajthom90/sonarr2/internal/providers/notification/synology"
	"github.com/ajthom90/sonarr2/internal/providers/notification/telegram"
	"github.com/ajthom90/sonarr2/internal/providers/notification/trakt"
	"github.com/ajthom90/sonarr2/internal/providers/notification/twitter"
	notifwebhook "github.com/ajthom90/sonarr2/internal/providers/notification/webhook"
	"github.com/ajthom90/sonarr2/internal/realtime"
	"github.com/ajthom90/sonarr2/internal/releaseprofile"
	"github.com/ajthom90/sonarr2/internal/remotepathmapping"
	"github.com/ajthom90/sonarr2/internal/rsssync"
	"github.com/ajthom90/sonarr2/internal/scheduler"
	"github.com/ajthom90/sonarr2/internal/tags"
	"github.com/ajthom90/sonarr2/internal/updatecheck"
)

// App is the running sonarr2 process.
type App struct {
	log             *slog.Logger
	server          *api.Server
	pool            db.Pool
	bus             events.Bus
	broker          *realtime.Broker
	library         *library.Library
	cmdQueue        commands.Queue
	registry        *commands.Registry
	workers         *commands.WorkerPool
	scheduler       *scheduler.Scheduler
	qualityDefs     profiles.QualityDefinitionStore
	qualityProfiles profiles.QualityProfileStore
	customFormats   customformats.Store
	tagStore        tags.Store
	blocklistStore  blocklist.Store
	rpmStore        remotepathmapping.Store
	releaseProfiles releaseprofile.Store
	delayProfiles   delayprofile.Store
	indexerRegistry *indexer.Registry
	dcRegistry      *downloadclient.Registry
	notifRegistry   *notification.Registry
	indexerStore    indexer.InstanceStore
	dcStore         downloadclient.InstanceStore
	notifStore      notification.InstanceStore
	metadataSource  metadatasource.MetadataSource
	historyStore    history.Store
	grabService     *grab.Service
	engine          *decisionengine.Engine
	fsWatcher       *fswatcher.Watcher
	checker         *health.Checker
}

// New constructs an App from the given config. It opens the database,
// runs migrations, seeds default config, creates the event bus and
// library stores, wires stats recompute subscribers, and builds the
// HTTP server.
func New(ctx context.Context, cfg config.Config) (*App, error) {
	log := logging.New(cfg.Logging, os.Stderr)

	pool, err := db.OpenFromConfig(ctx, cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("app: open db: %w", err)
	}

	if err := db.Migrate(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: migrate db: %w", err)
	}

	if err := seedHostConfig(ctx, pool); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: seed host config: %w", err)
	}

	// Read back the auth mode for the AuthCheck health check.
	authMode := "forms"
	{
		var hcStoreEarly hostconfig.Store
		switch p := pool.(type) {
		case *db.PostgresPool:
			hcStoreEarly = hostconfig.NewPostgresStore(p)
		case *db.SQLitePool:
			hcStoreEarly = hostconfig.NewSQLiteStore(p)
		}
		if hcStoreEarly != nil {
			if hc, err := hcStoreEarly.Get(ctx); err == nil && hc.AuthMode != "" {
				authMode = hc.AuthMode
			}
		}
	}

	bus := events.NewBus(16)

	rtBroker := realtime.NewBroker(256)
	rtBroker.SubscribeToEvents(bus)

	lib, err := library.New(pool, bus)
	if err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: library new: %w", err)
	}

	// Wire statistics recompute subscribers. Any time episodes or
	// episode files change for a series, recompute the cached stats row
	// for that series synchronously so the next read reflects reality.
	events.SubscribeSync[library.EpisodeAdded](bus, func(ctx context.Context, e library.EpisodeAdded) error {
		return lib.Stats.Recompute(ctx, e.SeriesID)
	})
	events.SubscribeSync[library.EpisodeUpdated](bus, func(ctx context.Context, e library.EpisodeUpdated) error {
		return lib.Stats.Recompute(ctx, e.SeriesID)
	})
	events.SubscribeSync[library.EpisodeDeleted](bus, func(ctx context.Context, e library.EpisodeDeleted) error {
		return lib.Stats.Recompute(ctx, e.SeriesID)
	})
	events.SubscribeSync[library.EpisodeFileAdded](bus, func(ctx context.Context, e library.EpisodeFileAdded) error {
		return lib.Stats.Recompute(ctx, e.SeriesID)
	})
	events.SubscribeSync[library.EpisodeFileDeleted](bus, func(ctx context.Context, e library.EpisodeFileDeleted) error {
		return lib.Stats.Recompute(ctx, e.SeriesID)
	})
	events.SubscribeSync[library.SeriesDeleted](bus, func(ctx context.Context, e library.SeriesDeleted) error {
		return lib.Stats.Delete(ctx, e.ID)
	})

	// Create profile, custom format, and tag stores.
	var qualityDefStore profiles.QualityDefinitionStore
	var qualityProfileStore profiles.QualityProfileStore
	var cfStore customformats.Store
	var tagStore tags.Store
	var blStore blocklist.Store
	var rpmStore remotepathmapping.Store
	var rpStore releaseprofile.Store
	var dpStore delayprofile.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		qualityDefStore = profiles.NewPostgresQualityDefinitionStore(p)
		qualityProfileStore = profiles.NewPostgresQualityProfileStore(p)
		cfStore = customformats.NewPostgresStore(p)
		tagStore = tags.NewPostgresStore(p)
		blStore = blocklist.NewPostgresStore(p)
		rpmStore = remotepathmapping.NewPostgresStore(p)
		rpStore = releaseprofile.NewPostgresStore(p)
		dpStore = delayprofile.NewPostgresStore(p)
	case *db.SQLitePool:
		qualityDefStore = profiles.NewSQLiteQualityDefinitionStore(p)
		qualityProfileStore = profiles.NewSQLiteQualityProfileStore(p)
		cfStore = customformats.NewSQLiteStore(p)
		tagStore = tags.NewSQLiteStore(p)
		blStore = blocklist.NewSQLiteStore(p)
		rpmStore = remotepathmapping.NewSQLiteStore(p)
		rpStore = releaseprofile.NewSQLiteStore(p)
		dpStore = delayprofile.NewSQLiteStore(p)
	default:
		_ = pool.Close()
		return nil, fmt.Errorf("app: unsupported pool type for profiles/CF: %T", pool)
	}

	// Seed a default "Any" quality profile that allows all qualities if none exist.
	existing, err := qualityProfileStore.List(ctx)
	if err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: list quality profiles: %w", err)
	}
	if len(existing) == 0 {
		allDefs, err := qualityDefStore.GetAll(ctx)
		if err != nil {
			_ = pool.Close()
			return nil, fmt.Errorf("app: get quality definitions: %w", err)
		}
		items := make([]profiles.QualityProfileItem, 0, len(allDefs))
		for _, d := range allDefs {
			items = append(items, profiles.QualityProfileItem{
				QualityID: d.ID,
				Allowed:   true,
			})
		}
		_, err = qualityProfileStore.Create(ctx, profiles.QualityProfile{
			Name:           "Any",
			UpgradeAllowed: true,
			Cutoff:         0, // no cutoff
			Items:          items,
		})
		if err != nil {
			_ = pool.Close()
			return nil, fmt.Errorf("app: seed default quality profile: %w", err)
		}
	}

	// Provider registries.
	idxReg := indexer.NewRegistry()
	dcReg := downloadclient.NewRegistry()
	notifReg := notification.NewRegistry()

	// Register built-in indexer providers.
	idxReg.Register("Newznab", func() indexer.Indexer {
		return newznab.New(newznab.Settings{ApiPath: "/api"}, nil)
	})
	idxReg.Register("Torznab", func() indexer.Indexer {
		return torznab.New(torznab.Settings{ApiPath: "/api", MinSeeders: 1}, nil)
	})
	idxReg.Register("TorrentRss", func() indexer.Indexer {
		return torrentrss.New(torrentrss.Settings{}, nil)
	})
	idxReg.Register("IPTorrents", func() indexer.Indexer {
		return iptorrents.New(iptorrents.Settings{}, nil)
	})
	idxReg.Register("Nyaa", func() indexer.Indexer {
		return nyaa.New(nyaa.Settings{BaseURL: "https://nyaa.si"}, nil)
	})
	idxReg.Register("BroadcastheNet", func() indexer.Indexer {
		return broadcasthenet.New(broadcasthenet.Settings{}, nil)
	})

	// Register built-in download client providers.
	dcReg.Register("SABnzbd", func() downloadclient.DownloadClient {
		return sabnzbd.New(sabnzbd.Settings{Host: "localhost", Port: 8080, Category: "tv"}, nil)
	})
	dcReg.Register("NZBGet", func() downloadclient.DownloadClient {
		return nzbget.New(nzbget.Settings{Host: "localhost", Port: 6789, Category: "tv"}, nil)
	})
	dcReg.Register("qBittorrent", func() downloadclient.DownloadClient {
		return qbittorrent.New(qbittorrent.Settings{Host: "localhost", Port: 8080, Category: "tv"}, nil)
	})
	dcReg.Register("Transmission", func() downloadclient.DownloadClient {
		return transmission.New(transmission.Settings{Host: "localhost", Port: 9091, UrlBase: "/transmission/"}, nil)
	})
	dcReg.Register("Deluge", func() downloadclient.DownloadClient {
		return deluge.New(deluge.Settings{Host: "localhost", Port: 8112}, nil)
	})
	dcReg.Register("Blackhole", func() downloadclient.DownloadClient {
		return blackhole.New(blackhole.Settings{}, nil)
	})
	dcReg.Register("UsenetBlackhole", func() downloadclient.DownloadClient {
		return blackholeusenet.New(blackholeusenet.Settings{}, nil)
	})
	dcReg.Register("TorrentBlackhole", func() downloadclient.DownloadClient {
		return blackholetorrent.New(blackholetorrent.Settings{}, nil)
	})
	dcReg.Register("Aria2", func() downloadclient.DownloadClient {
		return aria2.New(aria2.Settings{}, nil)
	})
	dcReg.Register("NzbVortex", func() downloadclient.DownloadClient {
		return nzbvortex.New(nzbvortex.Settings{}, nil)
	})
	dcReg.Register("Pneumatic", func() downloadclient.DownloadClient {
		return pneumatic.New(pneumatic.Settings{}, nil)
	})
	dcReg.Register("DownloadStation", func() downloadclient.DownloadClient {
		return dstation.NewTorrent(dstation.Settings{}, nil)
	})
	dcReg.Register("UsenetDownloadStation", func() downloadclient.DownloadClient {
		return dstation.NewUsenet(dstation.Settings{}, nil)
	})
	dcReg.Register("RTorrent", func() downloadclient.DownloadClient {
		return rtorrent.New(rtorrent.Settings{}, nil)
	})
	dcReg.Register("UTorrent", func() downloadclient.DownloadClient {
		return utorrent.New(utorrent.Settings{}, nil)
	})
	dcReg.Register("Vuze", func() downloadclient.DownloadClient {
		return vuze.New(vuze.Settings{}, nil)
	})
	dcReg.Register("Hadouken", func() downloadclient.DownloadClient {
		return hadouken.New(hadouken.Settings{}, nil)
	})
	dcReg.Register("Flood", func() downloadclient.DownloadClient {
		return flood.New(flood.Settings{}, nil)
	})
	dcReg.Register("FreeboxDownload", func() downloadclient.DownloadClient {
		return freebox.New(freebox.Settings{}, nil)
	})
	dcReg.Register("Tribler", func() downloadclient.DownloadClient {
		return tribler.New(tribler.Settings{}, nil)
	})
	dcReg.Register("RQBit", func() downloadclient.DownloadClient {
		return rqbit.New(rqbit.Settings{}, nil)
	})

	// Register built-in notification providers.
	notifReg.Register("Discord", func() notification.Notification {
		return discord.New(discord.Settings{}, nil)
	})
	notifReg.Register("Slack", func() notification.Notification {
		return slack.New(slack.Settings{}, nil)
	})
	notifReg.Register("Telegram", func() notification.Notification {
		return telegram.New(telegram.Settings{}, nil)
	})
	notifReg.Register("Email", func() notification.Notification {
		return notifyemail.New(notifyemail.Settings{})
	})
	notifReg.Register("Webhook", func() notification.Notification {
		return notifwebhook.New(notifwebhook.Settings{Method: "POST"}, nil)
	})
	notifReg.Register("Pushover", func() notification.Notification {
		return pushover.New(pushover.Settings{}, nil)
	})
	notifReg.Register("Gotify", func() notification.Notification {
		return gotify.New(gotify.Settings{}, nil)
	})
	notifReg.Register("CustomScript", func() notification.Notification {
		return customscript.New(customscript.Settings{})
	})
	notifReg.Register("PushBullet", func() notification.Notification {
		return pushbullet.New(pushbullet.Settings{}, nil)
	})
	notifReg.Register("Ntfy", func() notification.Notification {
		return ntfy.New(ntfy.Settings{}, nil)
	})
	notifReg.Register("Xbmc", func() notification.Notification {
		return kodi.New(kodi.Settings{}, nil)
	})
	notifReg.Register("PlexServer", func() notification.Notification {
		return plex.New(plex.Settings{}, nil)
	})
	notifReg.Register("MediaBrowser", func() notification.Notification {
		return emby.New(emby.Settings{}, nil)
	})
	notifReg.Register("Notifiarr", func() notification.Notification {
		return notifiarr.New(notifiarr.Settings{}, nil)
	})
	notifReg.Register("Prowl", func() notification.Notification {
		return prowl.New(prowl.Settings{}, nil)
	})
	notifReg.Register("Apprise", func() notification.Notification {
		return apprise.New(apprise.Settings{}, nil)
	})
	notifReg.Register("Join", func() notification.Notification {
		return notifyjoin.New(notifyjoin.Settings{}, nil)
	})
	notifReg.Register("Simplepush", func() notification.Notification {
		return simplepush.New(simplepush.Settings{}, nil)
	})
	notifReg.Register("Pushcut", func() notification.Notification {
		return pushcut.New(pushcut.Settings{}, nil)
	})
	notifReg.Register("Mailgun", func() notification.Notification {
		return mailgun.New(mailgun.Settings{}, nil)
	})
	notifReg.Register("SendGrid", func() notification.Notification {
		return sendgrid.New(sendgrid.Settings{}, nil)
	})
	notifReg.Register("Signal", func() notification.Notification {
		return notifysignal.New(notifysignal.Settings{}, nil)
	})
	notifReg.Register("Twitter", func() notification.Notification {
		return twitter.New(twitter.Settings{}, nil)
	})
	notifReg.Register("Trakt", func() notification.Notification {
		return trakt.New(trakt.Settings{}, nil)
	})
	notifReg.Register("SynologyIndexer", func() notification.Notification {
		return synology.New(synology.Settings{}, nil)
	})

	// Provider instance stores.
	var idxStore indexer.InstanceStore
	var dcStore downloadclient.InstanceStore
	var notifStore notification.InstanceStore
	switch p := pool.(type) {
	case *db.PostgresPool:
		idxStore = indexer.NewPostgresInstanceStore(p)
		dcStore = downloadclient.NewPostgresInstanceStore(p)
		notifStore = notification.NewPostgresInstanceStore(p)
	case *db.SQLitePool:
		idxStore = indexer.NewSQLiteInstanceStore(p)
		dcStore = downloadclient.NewSQLiteInstanceStore(p)
		notifStore = notification.NewSQLiteInstanceStore(p)
	}

	// Create the command queue (dialect-dispatched).
	var cmdQueue commands.Queue
	var taskStore scheduler.TaskStore
	switch p := pool.(type) {
	case *db.PostgresPool:
		cmdQueue = commands.NewPostgresQueue(p)
		taskStore = scheduler.NewPostgresTaskStore(p)
	case *db.SQLitePool:
		cmdQueue = commands.NewSQLiteQueue(p)
		taskStore = scheduler.NewSQLiteTaskStore(p)
	default:
		_ = pool.Close()
		return nil, fmt.Errorf("app: unsupported pool type %T for command queue", pool)
	}

	reg := commands.NewRegistry()
	numWorkers := runtime.NumCPU()
	if numWorkers > 4 {
		numWorkers = 4
	}
	wp := commands.NewWorkerPool(cmdQueue, reg, bus, log, numWorkers)
	sched := scheduler.New(taskStore, cmdQueue, log)

	// Create the TVDB metadata source with rate limiting and caching.
	// API key comes from config (env var SONARR2_TVDB_API_KEY or YAML).
	// The handler will return an error if called without a valid key.
	tvdbTransport := tvdb.NewRateLimitedTransport(http.DefaultTransport, tvdb.RateLimitOptions{
		RequestsPerSecond: cfg.TVDB.RateLimit,
		Burst:             cfg.TVDB.RateBurst,
		MaxRetries:        3,
	})
	tvdbClient := tvdb.New(tvdb.Settings{ApiKey: cfg.TVDB.ApiKey}, &http.Client{Transport: tvdbTransport})
	tvdbSource := cached.New(tvdbClient, cached.Options{
		SeriesTTL:   cfg.TVDB.CacheSeriesTTL,
		EpisodesTTL: cfg.TVDB.CacheEpisodesTTL,
		SearchTTL:   cfg.TVDB.CacheSearchTTL,
	})

	// Register built-in command handlers.
	cleanup := handlers.NewCleanupHandler(cmdQueue)
	reg.Register("MessagingCleanup", cleanup)

	refreshHandler := handlers.NewRefreshSeriesHandler(tvdbSource, lib)
	reg.Register("RefreshSeriesMetadata", refreshHandler)

	// Register the MessagingCleanup scheduled task (1-hour interval).
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "MessagingCleanup",
		IntervalSecs:  3600,
		NextExecution: time.Now().Add(time.Hour),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert MessagingCleanup task: %w", err)
	}

	// History store (dialect-dispatched).
	var histStore history.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		histStore = history.NewPostgresStore(p)
	case *db.SQLitePool:
		histStore = history.NewSQLiteStore(p)
	}

	// Import service — scans completed download folders and moves files into
	// the library. Registered as the "ProcessDownload" command handler so it
	// can be triggered via the command queue when a download completes.
	importSvc := importer.New(lib, histStore, bus, log)
	processDownload := handlers.NewProcessDownloadHandler(importSvc)
	reg.Register("ProcessDownload", processDownload)

	// ScanSeriesFolder handler — triggered by the filesystem watcher when files
	// change in a series folder.
	scanSeries := handlers.NewScanSeriesFolderHandler(lib, importSvc, log)
	reg.Register("ScanSeriesFolder", scanSeries)

	// RefreshMonitoredDownloads handler — polls download clients for completed
	// items and enqueues ProcessDownload for each new completion.
	refreshDownloads := handlers.NewRefreshMonitoredDownloadsHandler(
		dcStore, dcReg, cmdQueue, histStore, log,
	)
	reg.Register("RefreshMonitoredDownloads", refreshDownloads)

	// Schedule RefreshMonitoredDownloads at 1-minute interval.
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "RefreshMonitoredDownloads",
		IntervalSecs:  60,
		NextExecution: time.Now().Add(time.Minute),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert RefreshMonitoredDownloads task: %w", err)
	}

	// Filesystem watcher — monitors series root folders for new or changed
	// files and enqueues ScanSeriesFolder commands. Root folders are not
	// auto-added here; they will be registered via the API in M11. The watcher
	// is ready and its lifecycle (Start/Stop) is managed alongside the scheduler.
	resolver := &appSeriesResolver{library: lib}
	enqueuer := &queueEnqueuer{queue: cmdQueue}
	fsWatch := fswatcher.New(resolver, enqueuer, log)

	// Load quality definitions for specs that need them.
	allDefs, _ := qualityDefStore.GetAll(ctx)

	// Decision engine with 8 M5 specs.
	engine := decisionengine.New(
		specs.QualityAllowedSpec{},
		specs.CustomFormatScoreSpec{},
		specs.UpgradeAllowedSpec{},
		specs.UpgradableSpec{},
		specs.AcceptableSizeSpec{QualityDefs: allDefs},
		specs.NotSampleSpec{},
		specs.RepackSpec{},
		specs.AlreadyImportedSpec{},
	)

	// Grab service.
	grabSvc := grab.New(dcStore, dcReg, histStore, bus, log)

	// RSS sync handler.
	rssSyncHandler := rsssync.New(
		idxStore, idxReg, lib, engine, grabSvc,
		qualityDefStore, qualityProfileStore, cfStore, log,
	)
	reg.Register("RssSync", rssSyncHandler)

	// Register RssSync as a scheduled task (15-minute interval).
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "RssSync",
		IntervalSecs:  900,
		NextExecution: time.Now().Add(15 * time.Minute),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert RssSync task: %w", err)
	}

	// Subscribe history cleanup on SeriesDeleted.
	events.SubscribeSync[library.SeriesDeleted](bus, func(ctx context.Context, e library.SeriesDeleted) error {
		return histStore.DeleteForSeries(ctx, e.ID)
	})

	// Update checker — queries GitHub Releases API for newer versions.
	updateChecker := updatecheck.New(
		buildinfo.Get().Version,
		"ajthom90",
		"sonarr2",
		nil, // default HTTP client
	)

	// Health checks.
	checker := health.NewChecker(
		health.NewDatabaseCheck(pool),
		health.NewRootFolderCheck(&rootPathAdapter{series: lib.Series}),
		health.NewIndexerCheck(&indexerCountAdapter{store: idxStore}),
		health.NewDownloadClientCheck(&dcCountAdapter{store: dcStore}),
		health.NewMetadataSourceCheck(cfg.TVDB.ApiKey),
		health.NewAuthCheck(authMode),
		health.NewUpdateCheck(updateChecker),
	)

	// Run initial health check.
	checker.RunAll(ctx)

	// HealthCheck command handler — runs checks and dispatches notifications.
	healthHandler := &healthCheckHandler{
		checker:    checker,
		notifStore: notifStore,
		notifReg:   notifReg,
		log:        log,
	}
	reg.Register("HealthCheck", healthHandler)

	// Schedule HealthCheck at 30-minute interval.
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "HealthCheck",
		IntervalSecs:  1800,
		NextExecution: time.Now().Add(30 * time.Minute),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert HealthCheck task: %w", err)
	}

	// Housekeeping runner.
	housekeeper := housekeeping.New(housekeeping.Options{
		History:          histStore,
		EpisodeFiles:     &episodeFileAdapter{store: lib.EpisodeFiles},
		Series:           &seriesAdapter{store: lib.Series},
		Stats:            lib.Stats,
		DB:               &vacuumAdapter{pool: pool},
		Log:              log,
		HistoryRetention: cfg.HistoryRetention,
	})

	housekeepingHandler := &housekeepingTaskHandler{runner: housekeeper}
	reg.Register("Housekeeping", housekeepingHandler)

	// Schedule Housekeeping at 24-hour interval.
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "Housekeeping",
		IntervalSecs:  86400,
		NextExecution: time.Now().Add(24 * time.Hour),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert Housekeeping task: %w", err)
	}

	// Backup service.
	dbPath := ""
	if cfg.DB.Dialect == "sqlite" {
		dbPath = extractSQLitePath(cfg.DB.DSN)
	}
	backupSvc := backup.New(backup.Options{
		BackupDir:  filepath.Join(cfg.Paths.Config, "Backups"),
		DBPath:     dbPath,
		DBDialect:  cfg.DB.Dialect,
		AppVersion: buildinfo.Get().Version,
		Retention:  cfg.BackupRetention,
		Log:        log,
	})

	backupHandler := &backupTaskHandler{svc: backupSvc}
	reg.Register("Backup", backupHandler)

	// Schedule Backup at configured interval (default 7 days).
	if err := taskStore.Upsert(ctx, scheduler.ScheduledTask{
		TypeName:      "Backup",
		IntervalSecs:  int(cfg.BackupInterval.Seconds()),
		NextExecution: time.Now().Add(cfg.BackupInterval),
	}); err != nil {
		_ = pool.Close()
		return nil, fmt.Errorf("app: upsert Backup task: %w", err)
	}

	// Subscribe async notification dispatch on ReleasesGrabbed.
	events.SubscribeAsync[grab.ReleasesGrabbed](bus, func(ctx context.Context, e grab.ReleasesGrabbed) {
		dispatchGrabNotifications(ctx, notifStore, notifReg, log, notification.GrabMessage{
			SeriesTitle: e.Title,
		})
	})

	// Build host config store for API key auth.
	var hcStore hostconfig.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		hcStore = hostconfig.NewPostgresStore(p)
	case *db.SQLitePool:
		hcStore = hostconfig.NewSQLiteStore(p)
	}

	// Auth stores.
	var userStore auth.UserStore
	var sessionStore auth.SessionStore
	switch p := pool.(type) {
	case *db.PostgresPool:
		userStore = auth.NewPostgresUserStore(p)
		sessionStore = auth.NewPostgresSessionStore(p)
	case *db.SQLitePool:
		userStore = auth.NewSQLiteUserStore(p)
		sessionStore = auth.NewSQLiteSessionStore(p)
	}

	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log: log,
		server: api.NewWithDeps(addr, log, api.Deps{
			Pool:                 poolPingerAdapter{pool: pool},
			HostConfig:           hcStore,
			Series:               lib.Series,
			Seasons:              lib.Seasons,
			Stats:                lib.Stats,
			Episodes:             lib.Episodes,
			EpisodeFiles:         lib.EpisodeFiles,
			QualityProfiles:      qualityProfileStore,
			QualityDefs:          qualityDefStore,
			CustomFormats:        cfStore,
			Tags:                 tagStore,
			Blocklist:            blStore,
			RemotePathMappings:   rpmStore,
			ReleaseProfiles:      rpStore,
			DelayProfiles:        dpStore,
			Commands:             cmdQueue,
			History:              histStore,
			IndexerStore:         idxStore,
			URLBase:              cfg.HTTP.URLBase,
			RateLimit:            cfg.APIRateLimit,
			RateBurst:            cfg.APIRateBurst,
			IndexerRegistry:      idxReg,
			DCStore:              dcStore,
			DCRegistry:           dcReg,
			NotificationStore:    notifStore,
			NotificationRegistry: notifReg,
			MetadataSource:       tvdbSource,
			UserStore:            userStore,
			SessionStore:         sessionStore,
			HealthChecker:        checker,
			Broker:               rtBroker,
			BackupService:        backupSvc,
			OnTvdbKeyChanged:     func(key string) { tvdbClient.SetApiKey(key) },
			Log:                  log,
		}),
		pool:            pool,
		bus:             bus,
		broker:          rtBroker,
		library:         lib,
		cmdQueue:        cmdQueue,
		registry:        reg,
		workers:         wp,
		scheduler:       sched,
		qualityDefs:     qualityDefStore,
		qualityProfiles: qualityProfileStore,
		customFormats:   cfStore,
		tagStore:        tagStore,
		blocklistStore:  blStore,
		rpmStore:        rpmStore,
		releaseProfiles: rpStore,
		delayProfiles:   dpStore,
		indexerRegistry: idxReg,
		dcRegistry:      dcReg,
		notifRegistry:   notifReg,
		indexerStore:    idxStore,
		dcStore:         dcStore,
		notifStore:      notifStore,
		metadataSource:  tvdbSource,
		historyStore:    histStore,
		grabService:     grabSvc,
		engine:          engine,
		fsWatcher:       fsWatch,
		checker:         checker,
	}, nil
}

// seedHostConfig inserts a default host_config row with a freshly generated
// API key if none exists. Called once on startup from New.
func seedHostConfig(ctx context.Context, pool db.Pool) error {
	var store hostconfig.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		store = hostconfig.NewPostgresStore(p)
	case *db.SQLitePool:
		store = hostconfig.NewSQLiteStore(p)
	default:
		return fmt.Errorf("app: unsupported pool type %T", pool)
	}

	_, err := store.Get(ctx)
	if err == nil {
		return nil // already seeded
	}
	if !errors.Is(err, hostconfig.ErrNotFound) {
		return fmt.Errorf("app: get host config: %w", err)
	}

	return store.Upsert(ctx, hostconfig.HostConfig{
		APIKey:         hostconfig.NewAPIKey(),
		AuthMode:       "forms",
		MigrationState: "clean",
	})
}

// Run starts the HTTP server and blocks until ctx is cancelled or the server
// errors. It then performs a graceful shutdown with a 30s deadline.
func (a *App) Run(ctx context.Context) error {
	info := buildinfo.Get()
	a.log.Info("sonarr2 starting",
		slog.String("version", info.Version),
		slog.String("commit", info.Commit),
		slog.String("date", info.Date),
	)

	a.workers.Start(ctx)
	a.scheduler.Start(ctx)
	// Filesystem watcher is started here. Root folders are registered
	// dynamically via the API (M11+). Stop is called in the shutdown block.
	_ = a.fsWatcher // watcher is ready; no root folders to add at startup

	errCh := make(chan error, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.server.Start(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	wg.Wait()

	// Surface any Start error that arrived concurrently with ctx cancellation.
	select {
	case err := <-errCh:
		return fmt.Errorf("server failed: %w", err)
	default:
	}

	a.fsWatcher.Stop()
	a.scheduler.Stop()
	a.workers.Stop()

	if err := a.pool.Close(); err != nil {
		a.log.Error("db close error", slog.String("err", err.Error()))
	}

	a.log.Info("sonarr2 stopped")
	return nil
}

// SignalContext returns a context that cancels on SIGINT or SIGTERM, or when
// parent is cancelled.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}

// poolPingerAdapter wraps a db.Pool to satisfy api.PoolPinger by returning
// the dialect as a plain string. This keeps the api package free of a
// db-package import.
type poolPingerAdapter struct {
	pool db.Pool
}

func (p poolPingerAdapter) Dialect() string                { return string(p.pool.Dialect()) }
func (p poolPingerAdapter) Ping(ctx context.Context) error { return p.pool.Ping(ctx) }

// appSeriesResolver implements fswatcher.SeriesResolver by mapping filesystem
// paths to library series IDs using a prefix match.
type appSeriesResolver struct {
	library *library.Library
}

func (r *appSeriesResolver) ResolveSeriesID(path string) (int64, bool) {
	all, err := r.library.Series.List(context.Background())
	if err != nil {
		return 0, false
	}
	for _, s := range all {
		if strings.HasPrefix(path, s.Path) {
			return s.ID, true
		}
	}
	return 0, false
}

// queueEnqueuer implements fswatcher.CommandEnqueuer by wrapping commands.Queue.
type queueEnqueuer struct {
	queue commands.Queue
}

func (q *queueEnqueuer) Enqueue(ctx context.Context, name string, body []byte) error {
	_, err := q.queue.Enqueue(ctx, name, body, commands.PriorityNormal, commands.TriggerScheduled, "")
	return err
}

// rootPathAdapter adapts library.SeriesStore for health.SeriesPathLister.
type rootPathAdapter struct {
	series library.SeriesStore
}

func (a *rootPathAdapter) ListRootPaths(ctx context.Context) ([]string, error) {
	all, err := a.series.List(ctx)
	if err != nil {
		return nil, err
	}
	paths := make([]string, len(all))
	for i, s := range all {
		paths[i] = s.Path
	}
	return paths, nil
}

// indexerCountAdapter adapts indexer.InstanceStore for health.EnabledCounter.
type indexerCountAdapter struct {
	store indexer.InstanceStore
}

func (a *indexerCountAdapter) CountEnabled(ctx context.Context) (int, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, inst := range all {
		if inst.EnableRss || inst.EnableAutomaticSearch || inst.EnableInteractiveSearch {
			n++
		}
	}
	return n, nil
}

// dcCountAdapter adapts downloadclient.InstanceStore for health.EnabledCounter.
type dcCountAdapter struct {
	store downloadclient.InstanceStore
}

func (a *dcCountAdapter) CountEnabled(ctx context.Context) (int, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, inst := range all {
		if inst.Enable {
			n++
		}
	}
	return n, nil
}

// healthCheckHandler runs health checks and dispatches notifications for new issues.
type healthCheckHandler struct {
	checker    *health.Checker
	notifStore notification.InstanceStore
	notifReg   *notification.Registry
	log        *slog.Logger
	lastIssues map[string]bool
}

func (h *healthCheckHandler) Handle(ctx context.Context, _ commands.Command) error {
	results := h.checker.RunAll(ctx)

	currentIssues := map[string]bool{}
	for _, r := range results {
		if r.Type == health.LevelWarning || r.Type == health.LevelError {
			key := r.Source + ":" + r.Message
			currentIssues[key] = true
			if h.lastIssues != nil && h.lastIssues[key] {
				continue // already reported
			}
			dispatchHealthNotifications(ctx, h.notifStore, h.notifReg, h.log, notification.HealthMessage{
				Type:    string(r.Type),
				Message: r.Message,
			})
		}
	}
	h.lastIssues = currentIssues
	return nil
}

// dispatchHealthNotifications sends health issue notifications to enabled providers.
func dispatchHealthNotifications(
	ctx context.Context,
	store notification.InstanceStore,
	reg *notification.Registry,
	log *slog.Logger,
	msg notification.HealthMessage,
) {
	instances, err := store.List(ctx)
	if err != nil {
		log.Error("health notification dispatch: list instances", slog.String("err", err.Error()))
		return
	}
	for _, inst := range instances {
		if !inst.OnHealthIssue {
			continue
		}
		factory, err := reg.Get(inst.Implementation)
		if err != nil {
			continue
		}
		provider := factory()
		if err := provider.OnHealthIssue(ctx, msg); err != nil {
			log.Error("health notification dispatch: OnHealthIssue failed",
				slog.String("name", inst.Name),
				slog.String("err", err.Error()),
			)
		}
	}
}

// dispatchGrabNotifications loads all enabled notification instances that have
// OnGrab=true, instantiates each via the registry, and calls OnGrab. Errors
// are logged but do not stop dispatch to other providers.
func dispatchGrabNotifications(
	ctx context.Context,
	store notification.InstanceStore,
	reg *notification.Registry,
	log *slog.Logger,
	msg notification.GrabMessage,
) {
	instances, err := store.List(ctx)
	if err != nil {
		log.Error("notification dispatch: list instances", slog.String("err", err.Error()))
		return
	}
	for _, inst := range instances {
		if !inst.OnGrab {
			continue
		}
		factory, err := reg.Get(inst.Implementation)
		if err != nil {
			log.Warn("notification dispatch: unknown implementation",
				slog.String("implementation", inst.Implementation),
				slog.String("name", inst.Name),
			)
			continue
		}
		provider := factory()
		if err := provider.OnGrab(ctx, msg); err != nil {
			log.Error("notification dispatch: OnGrab failed",
				slog.String("name", inst.Name),
				slog.String("implementation", inst.Implementation),
				slog.String("err", err.Error()),
			)
		}
	}
}

// episodeFileAdapter adapts library.EpisodeFilesStore for housekeeping.
type episodeFileAdapter struct {
	store library.EpisodeFilesStore
}

func (a *episodeFileAdapter) ListForSeries(ctx context.Context, seriesID int64) ([]housekeeping.EpisodeFileInfo, error) {
	files, err := a.store.ListForSeries(ctx, seriesID)
	if err != nil {
		return nil, err
	}
	result := make([]housekeeping.EpisodeFileInfo, len(files))
	for i, f := range files {
		result[i] = housekeeping.EpisodeFileInfo{
			ID:           f.ID,
			SeriesID:     f.SeriesID,
			RelativePath: f.RelativePath,
		}
	}
	return result, nil
}

func (a *episodeFileAdapter) Delete(ctx context.Context, id int64) error {
	return a.store.Delete(ctx, id)
}

// seriesAdapter adapts library.SeriesStore for housekeeping.
type seriesAdapter struct {
	store library.SeriesStore
}

func (a *seriesAdapter) ListAll(ctx context.Context) ([]housekeeping.SeriesInfo, error) {
	all, err := a.store.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]housekeeping.SeriesInfo, len(all))
	for i, s := range all {
		result[i] = housekeeping.SeriesInfo{ID: s.ID, Path: s.Path}
	}
	return result, nil
}

// vacuumAdapter adapts db.Pool for housekeeping.Vacuumer.
type vacuumAdapter struct {
	pool db.Pool
}

func (a *vacuumAdapter) Vacuum(ctx context.Context) error {
	switch p := a.pool.(type) {
	case *db.SQLitePool:
		return p.Vacuum(ctx)
	case *db.PostgresPool:
		return p.Vacuum(ctx)
	default:
		return nil
	}
}

// housekeepingTaskHandler wraps the Runner as a command handler.
type housekeepingTaskHandler struct {
	runner *housekeeping.Runner
}

func (h *housekeepingTaskHandler) Handle(ctx context.Context, _ commands.Command) error {
	h.runner.Run(ctx)
	return nil
}

// extractSQLitePath extracts the filesystem path from a SQLite DSN.
func extractSQLitePath(dsn string) string {
	path := strings.TrimPrefix(dsn, "file:")
	if idx := strings.Index(path, "?"); idx != -1 {
		path = path[:idx]
	}
	if path == ":memory:" || path == "" {
		return ""
	}
	return path
}

// backupTaskHandler wraps the backup Service as a command handler.
type backupTaskHandler struct {
	svc *backup.Service
}

func (h *backupTaskHandler) Handle(ctx context.Context, _ commands.Command) error {
	_, err := h.svc.Create(ctx)
	return err
}
