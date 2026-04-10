// Package app is the composition root for sonarr2 — it wires the logger,
// HTTP server, and graceful shutdown together.
package app

import (
	"context"
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
	"github.com/ajthom90/sonarr2/internal/logging"
)

// App is the running sonarr2 process.
type App struct {
	log    *slog.Logger
	server *api.Server
}

// New constructs an App from the given config. It creates the logger and
// server but does not start any goroutines.
func New(cfg config.Config) *App {
	log := logging.New(cfg.Logging, os.Stderr)
	addr := net.JoinHostPort(cfg.HTTP.BindAddress, strconv.Itoa(cfg.HTTP.Port))
	return &App{
		log:    log,
		server: api.New(addr, log),
	}
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

	a.log.Info("sonarr2 stopped")
	return nil
}

// SignalContext returns a context that cancels on SIGINT or SIGTERM, or when
// parent is cancelled.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
}
