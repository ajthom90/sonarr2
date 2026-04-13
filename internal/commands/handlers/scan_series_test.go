package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/importer"
	"github.com/ajthom90/sonarr2/internal/library"
)

// slogDiscard returns a slog.Logger that discards all output.
func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

// setupScanSeriesEnv creates an in-memory SQLite database and returns a
// ScanSeriesFolderHandler wired to a fully migrated library.
func setupScanSeriesEnv(t *testing.T) (*ScanSeriesFolderHandler, *library.Library) {
	t.Helper()
	ctx := context.Background()

	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{
		DSN:         ":memory:",
		BusyTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	bus := events.NewBus(4)
	lib, err := library.New(pool, bus)
	if err != nil {
		t.Fatalf("library.New: %v", err)
	}

	hist := history.NewSQLiteStore(pool)
	log := slogDiscard()
	svc := importer.New(lib, hist, bus, log)

	return NewScanSeriesFolderHandler(lib, svc, log), lib
}

// makeScanSeriesFolderCmd builds a ScanSeriesFolder command with the given seriesID.
func makeScanSeriesFolderCmd(t *testing.T, seriesID int64) commands.Command {
	t.Helper()
	body, err := json.Marshal(map[string]any{"seriesId": seriesID})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return commands.Command{Name: "ScanSeriesFolder", Body: body}
}

// TestScanSeriesFolderHandlerImportsFile is an end-to-end integration test:
// creates a series whose path is a temp dir containing a .mkv file, runs
// ScanSeriesFolderHandler, and verifies the file was imported into the library.
func TestScanSeriesFolderHandlerImportsFile(t *testing.T) {
	handler, lib := setupScanSeriesEnv(t)
	ctx := context.Background()

	// The series path doubles as the directory to scan.
	seriesDir := t.TempDir()

	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID:     6000,
		Title:      "Breaking Bad",
		Slug:       "breaking-bad",
		Status:     "ended",
		SeriesType: "standard",
		Path:       seriesDir,
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	ep, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "Pilot",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	// Place a large fake .mkv in the series dir (importer reads from series.Path).
	mediaPath := filepath.Join(seriesDir, "Breaking.Bad.S01E01.1080p.WEB-DL.mkv")
	createFakeMediaFile(t, mediaPath, 50*1024*1024) // 50 MB

	cmd := makeScanSeriesFolderCmd(t, series.ID)
	if err := handler.Handle(ctx, cmd); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify the file was moved into Season 01 inside the series dir.
	seasonDir := filepath.Join(seriesDir, "Season 01")
	entries, err := os.ReadDir(seasonDir)
	if err != nil {
		t.Fatalf("read season dir %q: %v", seasonDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("expected imported file in Season 01 dir; got none")
	}

	// Verify episode_files record.
	files, err := lib.EpisodeFiles.ListForSeries(ctx, series.ID)
	if err != nil {
		t.Fatalf("ListForSeries episode files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("episode_files count = %d, want 1", len(files))
	}

	// Verify episode.EpisodeFileID is set.
	updated, err := lib.Episodes.Get(ctx, ep.ID)
	if err != nil {
		t.Fatalf("get updated episode: %v", err)
	}
	if updated.EpisodeFileID == nil {
		t.Error("episode.EpisodeFileID should be non-nil after scan import")
	}
}

// TestScanSeriesFolderHandlerBadJSON verifies that invalid JSON body returns a
// parse error rather than panicking.
func TestScanSeriesFolderHandlerBadJSON(t *testing.T) {
	handler, _ := setupScanSeriesEnv(t)
	ctx := context.Background()

	cmd := commands.Command{
		Name: "ScanSeriesFolder",
		Body: []byte(`{not valid json`),
	}
	err := handler.Handle(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for bad JSON body; got nil")
	}
}
