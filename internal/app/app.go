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
	"github.com/ajthom90/sonarr2/internal/hostconfig"
	"github.com/ajthom90/sonarr2/internal/logging"
)

// App is the running sonarr2 process.
type App struct {
	log    *slog.Logger
	server *api.Server
	pool   db.Pool
}

// New constructs an App from the given config. It opens the database,
// runs migrations, and wires the HTTP server. If any step fails, returns
// the error. The caller is responsible for calling Run to start serving.
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

	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log:    log,
		server: api.New(addr, log),
		pool:   pool,
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
