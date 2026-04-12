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
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/ajthom90/sonarr2/internal/api"
	"github.com/ajthom90/sonarr2/internal/buildinfo"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/logging"
)

// App is the running sonarr2 process.
type App struct {
	log     *slog.Logger
	server  *api.Server
	pool    db.Pool
	bus     events.Bus
	library *library.Library
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

	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log:     log,
		server:  api.New(addr, log, poolPingerAdapter{pool: pool}),
		pool:    pool,
		bus:     bus,
		library: lib,
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
