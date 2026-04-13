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

// setupProcessDownloadEnv creates an in-memory SQLite database with migrations
// applied and returns a fully wired ProcessDownloadHandler plus supporting stores
// for assertions.
func setupProcessDownloadEnv(t *testing.T) (*ProcessDownloadHandler, *library.Library, history.Store) {
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
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	svc := importer.New(lib, hist, bus, log)

	return NewProcessDownloadHandler(svc), lib, hist
}

// makeProcessDownloadCmd builds a ProcessDownload command with the given body fields.
func makeProcessDownloadCmd(t *testing.T, downloadFolder string, seriesID int64, downloadID string) commands.Command {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"downloadFolder": downloadFolder,
		"seriesId":       seriesID,
		"downloadId":     downloadID,
	})
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	return commands.Command{Name: "ProcessDownload", Body: body}
}

// createFakeMediaFile creates a sparse media file of the given byte size.
func createFakeMediaFile(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %q: %v", path, err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatalf("truncate %q: %v", path, err)
	}
	f.Close()
}

// TestProcessDownloadHandlerImportsFile is the end-to-end integration test:
// create a temp dir with a fake .mkv, set up a series + episode in SQLite,
// run the handler with a command body, and verify the file was imported.
func TestProcessDownloadHandlerImportsFile(t *testing.T) {
	handler, lib, hist := setupProcessDownloadEnv(t)
	ctx := context.Background()

	// Seed series in library with a temp directory as its path.
	libraryDir := t.TempDir()
	series, err := lib.Series.Create(ctx, library.Series{
		TvdbID:     5000,
		Title:      "The Wire",
		Slug:       "the-wire",
		Status:     "ended",
		SeriesType: "standard",
		Path:       libraryDir,
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}

	ep, err := lib.Episodes.Create(ctx, library.Episode{
		SeriesID:      series.ID,
		SeasonNumber:  1,
		EpisodeNumber: 1,
		Title:         "The Target",
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}

	// Place a large enough .mkv in a download dir.
	downloadDir := t.TempDir()
	srcPath := filepath.Join(downloadDir, "The.Wire.S01E01.720p.WEB-DL.mkv")
	createFakeMediaFile(t, srcPath, 50*1024*1024) // 50 MB

	cmd := makeProcessDownloadCmd(t, downloadDir, series.ID, "dl-wire-001")
	if err := handler.Handle(ctx, cmd); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	// Verify destination file was created.
	dstDir := filepath.Join(libraryDir, "Season 01")
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatalf("read dest dir %q: %v", dstDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("expected imported file in Season 01 dir; got none")
	}

	// Verify episode_files record was created.
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
		t.Error("episode.EpisodeFileID should be non-nil after import")
	}

	// Verify history entry.
	entries2, err := hist.ListForEpisode(ctx, ep.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(entries2) == 0 {
		t.Fatal("expected history entry; got none")
	}
	if entries2[0].EventType != history.EventDownloadImported {
		t.Errorf("history event = %q, want %q", entries2[0].EventType, history.EventDownloadImported)
	}
	if entries2[0].DownloadID != "dl-wire-001" {
		t.Errorf("history download_id = %q, want dl-wire-001", entries2[0].DownloadID)
	}
}

// TestProcessDownloadHandlerBadJSON verifies that invalid JSON body returns a
// parse error rather than panicking.
func TestProcessDownloadHandlerBadJSON(t *testing.T) {
	handler, _, _ := setupProcessDownloadEnv(t)
	ctx := context.Background()

	cmd := commands.Command{
		Name: "ProcessDownload",
		Body: []byte(`{not valid json`),
	}
	err := handler.Handle(ctx, cmd)
	if err == nil {
		t.Fatal("expected error for bad JSON body; got nil")
	}
}
