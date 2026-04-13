// Package app is the composition root for sonarr2 — it wires the logger,
// HTTP server, and graceful shutdown together.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ajthom90/sonarr2/internal/api"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/commands/handlers"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/fswatcher"
	"github.com/ajthom90/sonarr2/internal/grab"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/importer"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/logging"
	"github.com/ajthom90/sonarr2/internal/profiles"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient/sabnzbd"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
	"github.com/ajthom90/sonarr2/internal/providers/indexer/newznab"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource/tvdb"
	"github.com/ajthom90/sonarr2/internal/rsssync"
	"github.com/ajthom90/sonarr2/internal/scheduler"
)

// App is the running sonarr2 process.
type App struct {
	log             *slog.Logger
	server          *api.Server
	pool            db.Pool
	bus             events.Bus
	library         *library.Library
	cmdQueue        commands.Queue
	registry        *commands.Registry
	workers         *commands.WorkerPool
	scheduler       *scheduler.Scheduler
	qualityDefs     profiles.QualityDefinitionStore
	qualityProfiles profiles.QualityProfileStore
	customFormats   customformats.Store
	indexerRegistry *indexer.Registry
	dcRegistry      *downloadclient.Registry
	indexerStore    indexer.InstanceStore
	dcStore         downloadclient.InstanceStore
	metadataSource  metadatasource.MetadataSource
	historyStore    history.Store
	grabService     *grab.Service
	engine          *decisionengine.Engine
	fsWatcher       *fswatcher.Watcher
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

	bus := events.NewBus(16)

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

	// Create profile and custom format stores.
	var qualityDefStore profiles.QualityDefinitionStore
	var qualityProfileStore profiles.QualityProfileStore
	var cfStore customformats.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		qualityDefStore = profiles.NewPostgresQualityDefinitionStore(p)
		qualityProfileStore = profiles.NewPostgresQualityProfileStore(p)
		cfStore = customformats.NewPostgresStore(p)
	case *db.SQLitePool:
		qualityDefStore = profiles.NewSQLiteQualityDefinitionStore(p)
		qualityProfileStore = profiles.NewSQLiteQualityProfileStore(p)
		cfStore = customformats.NewSQLiteStore(p)
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

	// Register built-in providers.
	idxReg.Register("Newznab", func() indexer.Indexer {
		return newznab.New(newznab.Settings{ApiPath: "/api"}, nil)
	})
	dcReg.Register("SABnzbd", func() downloadclient.DownloadClient {
		return sabnzbd.New(sabnzbd.Settings{Host: "localhost", Port: 8080, Category: "tv"}, nil)
	})

	// Provider instance stores.
	var idxStore indexer.InstanceStore
	var dcStore downloadclient.InstanceStore
	switch p := pool.(type) {
	case *db.PostgresPool:
		idxStore = indexer.NewPostgresInstanceStore(p)
		dcStore = downloadclient.NewPostgresInstanceStore(p)
	case *db.SQLitePool:
		idxStore = indexer.NewSQLiteInstanceStore(p)
		dcStore = downloadclient.NewSQLiteInstanceStore(p)
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

	// Create the TVDB metadata source. API key is empty by default — users
	// configure it via the UI or SONARR2_TVDB_API_KEY env var later. The
	// handler will return an error if called without a valid key.
	tvdbSource := tvdb.New(tvdb.Settings{ApiKey: ""}, nil)

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

	// Build host config store for API key auth.
	var hcStore hostconfig.Store
	switch p := pool.(type) {
	case *db.PostgresPool:
		hcStore = hostconfig.NewPostgresStore(p)
	case *db.SQLitePool:
		hcStore = hostconfig.NewSQLiteStore(p)
	}

	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log: log,
		server: api.NewWithDeps(addr, log, api.Deps{
			Pool:            poolPingerAdapter{pool: pool},
			HostConfig:      hcStore,
			Series:          lib.Series,
			Seasons:         lib.Seasons,
			Stats:           lib.Stats,
			Episodes:        lib.Episodes,
			EpisodeFiles:    lib.EpisodeFiles,
			QualityProfiles: qualityProfileStore,
			QualityDefs:     qualityDefStore,
			CustomFormats:   cfStore,
			Commands:        cmdQueue,
			History:         histStore,
			IndexerStore:    idxStore,
			IndexerRegistry: idxReg,
			DCStore:         dcStore,
			DCRegistry:      dcReg,
			Log:             log,
		}),
		pool:            pool,
		bus:             bus,
		library:         lib,
		cmdQueue:        cmdQueue,
		registry:        reg,
		workers:         wp,
		scheduler:       sched,
		qualityDefs:     qualityDefStore,
		qualityProfiles: qualityProfileStore,
		customFormats:   cfStore,
		indexerRegistry: idxReg,
		dcRegistry:      dcReg,
		indexerStore:    idxStore,
		dcStore:         dcStore,
		metadataSource:  tvdbSource,
		historyStore:    histStore,
		grabService:     grabSvc,
		engine:          engine,
		fsWatcher:       fsWatch,
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
