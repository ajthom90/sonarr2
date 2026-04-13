package importer_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/importer"
	"github.com/ajthom90/sonarr2/internal/library"
)

// testEnv bundles everything needed by importer tests.
type testEnv struct {
	lib  *library.Library
	hist history.Store
	bus  events.Bus
	svc  *importer.Service
	ctx  context.Context
}

func newTestEnv(t *testing.T) *testEnv {
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

	return &testEnv{lib: lib, hist: hist, bus: bus, svc: svc, ctx: ctx}
}

// createSeries creates a series with a temp directory as its Path.
func createSeries(t *testing.T, env *testEnv, title string) library.Series {
	t.Helper()
	libraryDir := t.TempDir()
	s, err := env.lib.Series.Create(env.ctx, library.Series{
		TvdbID:     1000,
		Title:      title,
		Slug:       "the-simpsons",
		Status:     "continuing",
		SeriesType: "standard",
		Path:       libraryDir,
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	return s
}

// createEpisode creates an episode for a series.
func createEpisode(t *testing.T, env *testEnv, seriesID int64, season, episode int32, title string) library.Episode {
	t.Helper()
	ep, err := env.lib.Episodes.Create(env.ctx, library.Episode{
		SeriesID:      seriesID,
		SeasonNumber:  season,
		EpisodeNumber: episode,
		Title:         title,
		Monitored:     true,
	})
	if err != nil {
		t.Fatalf("create episode: %v", err)
	}
	return ep
}

// createFakeFile creates a sparse file of the given size (bytes) at path.
// Uses Truncate so the file is created instantly without filling disk.
func createFakeFile(t *testing.T, path string, size int64) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create fake file %q: %v", path, err)
	}
	if err := f.Truncate(size); err != nil {
		f.Close()
		t.Fatalf("truncate fake file %q: %v", path, err)
	}
	f.Close()
}

// TestImportProcessFolderMovesFile verifies the happy path:
// a 50MB .mkv is moved to the library path, episode_file_id is set,
// and a history entry is recorded.
func TestImportProcessFolderMovesFile(t *testing.T) {
	env := newTestEnv(t)
	series := createSeries(t, env, "The Simpsons")
	ep := createEpisode(t, env, series.ID, 3, 5, "Homer the Heretic")

	downloadDir := t.TempDir()
	filename := "The.Simpsons.S03E05.720p.WEB-DL.mkv"
	srcPath := filepath.Join(downloadDir, filename)
	createFakeFile(t, srcPath, 50*1024*1024) // 50 MB

	if err := env.svc.ProcessFolder(env.ctx, downloadDir, series.ID, "dl-abc123"); err != nil {
		t.Fatalf("ProcessFolder: %v", err)
	}

	// Verify destination file exists.
	dstDir := filepath.Join(series.Path, "Season 03")
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatalf("read destination dir %q: %v", dstDir, err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one file in destination dir; got none")
	}

	// Verify episode_files record was created.
	files, err := env.lib.EpisodeFiles.ListForSeries(env.ctx, series.ID)
	if err != nil {
		t.Fatalf("ListForSeries episode files: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("episode_files count = %d, want 1", len(files))
	}

	// Verify episode.EpisodeFileID is set.
	updated, err := env.lib.Episodes.Get(env.ctx, ep.ID)
	if err != nil {
		t.Fatalf("get updated episode: %v", err)
	}
	if updated.EpisodeFileID == nil {
		t.Error("episode.EpisodeFileID should be non-nil after import")
	}

	// Verify history entry was created.
	hist, err := env.hist.ListForEpisode(env.ctx, ep.ID)
	if err != nil {
		t.Fatalf("list history for episode: %v", err)
	}
	if len(hist) == 0 {
		t.Fatal("expected history entry for imported episode; got none")
	}
	if hist[0].EventType != history.EventDownloadImported {
		t.Errorf("history event type = %q, want %q", hist[0].EventType, history.EventDownloadImported)
	}
	if hist[0].DownloadID != "dl-abc123" {
		t.Errorf("history download_id = %q, want dl-abc123", hist[0].DownloadID)
	}
}

// TestImportSkipsSampleFiles verifies that files under 40 MB are skipped.
func TestImportSkipsSampleFiles(t *testing.T) {
	env := newTestEnv(t)
	series := createSeries(t, env, "Breaking Bad")
	ep := createEpisode(t, env, series.ID, 1, 1, "Pilot")

	downloadDir := t.TempDir()
	srcPath := filepath.Join(downloadDir, "Breaking.Bad.S01E01.sample.mkv")
	createFakeFile(t, srcPath, 100) // 100 bytes — clearly a sample

	if err := env.svc.ProcessFolder(env.ctx, downloadDir, series.ID, "dl-sample"); err != nil {
		t.Fatalf("ProcessFolder: %v", err)
	}

	// No episode_file record.
	files, err := env.lib.EpisodeFiles.ListForSeries(env.ctx, series.ID)
	if err != nil {
		t.Fatalf("ListForSeries episode files: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("episode_files count = %d, want 0 (sample should be skipped)", len(files))
	}

	// No history.
	hist, err := env.hist.ListForEpisode(env.ctx, ep.ID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(hist) != 0 {
		t.Errorf("history entries = %d, want 0 (sample should be skipped)", len(hist))
	}
}

// TestImportSkipsNonMediaFiles verifies .txt and .nfo files are ignored.
func TestImportSkipsNonMediaFiles(t *testing.T) {
	env := newTestEnv(t)
	series := createSeries(t, env, "Seinfeld")
	_ = createEpisode(t, env, series.ID, 1, 1, "The Seinfeld Chronicles")

	downloadDir := t.TempDir()
	createFakeFile(t, filepath.Join(downloadDir, "readme.txt"), 50*1024*1024)
	createFakeFile(t, filepath.Join(downloadDir, "seinfeld.nfo"), 50*1024*1024)

	if err := env.svc.ProcessFolder(env.ctx, downloadDir, series.ID, "dl-nfo"); err != nil {
		t.Fatalf("ProcessFolder: %v", err)
	}

	files, err := env.lib.EpisodeFiles.ListForSeries(env.ctx, series.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("episode_files count = %d, want 0 (non-media files should be skipped)", len(files))
	}
}

// TestImportHardlinksWhenSameFS verifies that when source and destination are
// on the same filesystem, a hardlink is used: both paths point to the same
// inode, so the source file still exists after import.
func TestImportHardlinksWhenSameFS(t *testing.T) {
	env := newTestEnv(t)
	// Place the library directory inside the same temp tree so both src and dst
	// live on the same filesystem (guaranteed by t.TempDir).
	rootDir := t.TempDir()
	downloadDir := filepath.Join(rootDir, "download")
	if err := os.MkdirAll(downloadDir, 0o755); err != nil {
		t.Fatalf("mkdir download: %v", err)
	}
	libraryDir := filepath.Join(rootDir, "library")
	if err := os.MkdirAll(libraryDir, 0o755); err != nil {
		t.Fatalf("mkdir library: %v", err)
	}

	series, err := env.lib.Series.Create(env.ctx, library.Series{
		TvdbID:     2000,
		Title:      "Lost",
		Slug:       "lost",
		Status:     "ended",
		SeriesType: "standard",
		Path:       libraryDir,
		Monitored:  true,
	})
	if err != nil {
		t.Fatalf("create series: %v", err)
	}
	_ = createEpisode(t, env, series.ID, 1, 1, "Pilot")

	srcPath := filepath.Join(downloadDir, "Lost.S01E01.1080p.mkv")
	createFakeFile(t, srcPath, 50*1024*1024)

	if err := env.svc.ProcessFolder(env.ctx, downloadDir, series.ID, "dl-lost"); err != nil {
		t.Fatalf("ProcessFolder: %v", err)
	}

	// Source should still exist (hardlink does not remove the original).
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		t.Error("source file should still exist when a hardlink was used; got not-found")
	}

	// Destination should exist.
	dstDir := filepath.Join(libraryDir, "Season 01")
	entries, err := os.ReadDir(dstDir)
	if err != nil {
		t.Fatalf("read dest dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no files in destination dir after hardlink import")
	}

	// Both paths must share the same inode (proof of hardlink).
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		t.Fatalf("stat src: %v", err)
	}
	dstPath := filepath.Join(dstDir, entries[0].Name())
	dstInfo, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	srcSys, ok1 := srcInfo.Sys().(*syscall.Stat_t)
	dstSys, ok2 := dstInfo.Sys().(*syscall.Stat_t)
	if ok1 && ok2 {
		if srcSys.Ino != dstSys.Ino {
			t.Errorf("src inode %d != dst inode %d; expected hardlink", srcSys.Ino, dstSys.Ino)
		}
	}
}
