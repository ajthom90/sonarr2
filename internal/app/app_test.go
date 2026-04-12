package app

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/config"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/logging"
)

func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func TestAppRunAndShutdown(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError // quiet tests
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	a, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- a.Run(ctx)
	}()

	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Fatal("server did not start within 3s")
		}
		resp, err := http.Get(base + "/ping")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}

func TestSignalContextCancelsOnParentCancel(t *testing.T) {
	parent, parentCancel := context.WithCancel(context.Background())
	ctx, cancel := SignalContext(parent)
	defer cancel()

	parentCancel()

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(time.Second):
		t.Fatal("SignalContext did not cancel when parent was cancelled")
	}
}

func TestAppLibraryStatsRecompute(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	ctx := context.Background()
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		_ = a.pool.Close()
	})

	// Create a series.
	series, err := a.library.Series.Create(ctx, library.Series{
		TvdbID: 42, Title: "Test Show", Slug: "test-show",
		Status: "continuing", SeriesType: "standard",
		Path: "/tv/Test Show", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	// Create 3 episodes, 2 monitored.
	for i := int32(1); i <= 3; i++ {
		if _, err := a.library.Episodes.Create(ctx, library.Episode{
			SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: i,
			Title: "E", Monitored: i < 3,
		}); err != nil {
			t.Fatalf("Create episode: %v", err)
		}
	}

	// After creating episodes, the stats row should exist and reflect
	// the 3 episodes (2 monitored) via the EpisodeAdded subscriber.
	stats, err := a.library.Stats.Get(ctx, series.ID)
	if err != nil {
		t.Fatalf("Stats.Get after episodes: %v", err)
	}
	if stats.EpisodeCount != 3 {
		t.Errorf("EpisodeCount = %d, want 3", stats.EpisodeCount)
	}
	if stats.MonitoredEpisodeCount != 2 {
		t.Errorf("MonitoredEpisodeCount = %d, want 2", stats.MonitoredEpisodeCount)
	}
	if stats.EpisodeFileCount != 0 {
		t.Errorf("EpisodeFileCount = %d, want 0 before any files", stats.EpisodeFileCount)
	}

	// Create 2 episode files, 100 + 200 bytes.
	for _, size := range []int64{100, 200} {
		if _, err := a.library.EpisodeFiles.Create(ctx, library.EpisodeFile{
			SeriesID: series.ID, SeasonNumber: 1, RelativePath: "x.mkv", Size: size,
		}); err != nil {
			t.Fatalf("Create file: %v", err)
		}
	}

	// Stats should now reflect both episodes and files.
	stats, err = a.library.Stats.Get(ctx, series.ID)
	if err != nil {
		t.Fatalf("Stats.Get after files: %v", err)
	}
	if stats.EpisodeFileCount != 2 {
		t.Errorf("EpisodeFileCount = %d, want 2", stats.EpisodeFileCount)
	}
	if stats.SizeOnDisk != 300 {
		t.Errorf("SizeOnDisk = %d, want 300", stats.SizeOnDisk)
	}

	// Delete the series; Stats should be deleted via the SeriesDeleted subscriber.
	if err := a.library.Series.Delete(ctx, series.ID); err != nil {
		t.Fatalf("Delete series: %v", err)
	}
	_, err = a.library.Stats.Get(ctx, series.ID)
	if !errors.Is(err, library.ErrNotFound) {
		t.Errorf("Stats.Get after series delete error = %v, want ErrNotFound", err)
	}
}

func TestAppSeedsDefaultQualityProfile(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	ctx := context.Background()
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.pool.Close() })

	list, err := a.qualityProfiles.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 default profile, got %d", len(list))
	}
	if list[0].Name != "Any" {
		t.Errorf("Name = %q, want Any", list[0].Name)
	}
	if !list[0].UpgradeAllowed {
		t.Error("UpgradeAllowed = false, want true")
	}
	if len(list[0].Items) != 18 {
		t.Errorf("Items count = %d, want 18", len(list[0].Items))
	}
}

func TestAppProviderRegistriesExist(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	ctx := context.Background()
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = a.pool.Close() })

	_, err = a.indexerRegistry.Get("Newznab")
	if err != nil {
		t.Errorf("Newznab not registered: %v", err)
	}
	_, err = a.dcRegistry.Get("SABnzbd")
	if err != nil {
		t.Errorf("SABnzbd not registered: %v", err)
	}
	if a.indexerStore == nil {
		t.Error("indexerStore is nil")
	}
	if a.dcStore == nil {
		t.Error("dcStore is nil")
	}
}

func TestAppCommandExecution(t *testing.T) {
	port := findFreePort(t)
	cfg := config.Default()
	cfg.HTTP.Port = port
	cfg.HTTP.BindAddress = "127.0.0.1"
	cfg.Logging.Level = logging.LevelError
	cfg.DB.Dialect = "sqlite"
	cfg.DB.DSN = ":memory:"
	cfg.DB.BusyTimeout = 5 * time.Second

	ctx := context.Background()
	a, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Register a test handler.
	executed := make(chan struct{}, 1)
	a.registry.Register("TestCommand", commands.HandlerFunc(
		func(ctx context.Context, cmd commands.Command) error {
			executed <- struct{}{}
			return nil
		},
	))

	// Start the app's background systems.
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- a.Run(runCtx) }()

	// Wait for the HTTP server to be ready.
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if resp, err := http.Get(base + "/ping"); err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Enqueue a command.
	_, err = a.cmdQueue.Enqueue(ctx, "TestCommand", nil, commands.PriorityHigh, commands.TriggerManual, "")
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	// Wait for the handler to execute.
	select {
	case <-executed:
		// success
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not execute within 5s")
	}

	// Shut down.
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run() error = %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run() did not return after cancel")
	}
}
