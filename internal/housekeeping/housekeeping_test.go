package housekeeping

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- mock types ---

type mockHistory struct {
	deleteCalls int
	lastBefore  time.Time
	returnN     int64
	returnErr   error
}

func (m *mockHistory) DeleteBefore(_ context.Context, before time.Time) (int64, error) {
	m.deleteCalls++
	m.lastBefore = before
	return m.returnN, m.returnErr
}

type mockEpisodeFiles struct {
	files       map[int64][]EpisodeFileInfo // seriesID -> files
	deleteCalls []int64                     // IDs that were deleted
}

func (m *mockEpisodeFiles) ListForSeries(_ context.Context, seriesID int64) ([]EpisodeFileInfo, error) {
	return m.files[seriesID], nil
}

func (m *mockEpisodeFiles) Delete(_ context.Context, id int64) error {
	m.deleteCalls = append(m.deleteCalls, id)
	return nil
}

type mockSeries struct {
	series []SeriesInfo
}

func (m *mockSeries) ListAll(_ context.Context) ([]SeriesInfo, error) {
	return m.series, nil
}

type mockStats struct {
	recomputeCalls []int64
}

func (m *mockStats) Recompute(_ context.Context, seriesID int64) error {
	m.recomputeCalls = append(m.recomputeCalls, seriesID)
	return nil
}

type mockVacuumer struct {
	called bool
	err    error
}

func (m *mockVacuumer) Vacuum(_ context.Context) error {
	m.called = true
	return m.err
}

// errHistory always returns an error from DeleteBefore.
type errHistory struct{}

func (e *errHistory) DeleteBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, errSentinel
}

var errSentinel = &sentinelError{}

type sentinelError struct{}

func (s *sentinelError) Error() string { return "sentinel error" }

// discardLogger returns a slog.Logger that writes nothing.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- tests ---

func TestTrimHistory(t *testing.T) {
	hist := &mockHistory{returnN: 5}
	vac := &mockVacuumer{}
	stats := &mockStats{}
	series := &mockSeries{}
	files := &mockEpisodeFiles{files: map[int64][]EpisodeFileInfo{}}

	before := time.Now()
	r := New(Options{
		History:      hist,
		EpisodeFiles: files,
		Series:       series,
		Stats:        stats,
		DB:           vac,
		Log:          discardLogger(),
	})
	r.trimHistory(context.Background())
	after := time.Now()

	if hist.deleteCalls != 1 {
		t.Fatalf("expected DeleteBefore called once, got %d", hist.deleteCalls)
	}

	// cutoff should be approximately 90 days ago
	expected := before.Add(-90 * 24 * time.Hour)
	expectedEnd := after.Add(-90 * 24 * time.Hour)

	if hist.lastBefore.Before(expected.Add(-time.Second)) || hist.lastBefore.After(expectedEnd.Add(time.Second)) {
		t.Errorf("cutoff %v not within 1s of expected range [%v, %v]",
			hist.lastBefore, expected, expectedEnd)
	}
}

func TestCleanOrphanEpisodeFiles(t *testing.T) {
	dir := t.TempDir()

	// Create one real file on disk.
	realFile := "Season 01/episode01.mkv"
	realPath := filepath.Join(dir, realFile)
	if err := os.MkdirAll(filepath.Dir(realPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realPath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	const (
		seriesID   int64 = 42
		existingID int64 = 1
		missingID  int64 = 2
	)

	ef := &mockEpisodeFiles{
		files: map[int64][]EpisodeFileInfo{
			seriesID: {
				{ID: existingID, SeriesID: seriesID, RelativePath: realFile},
				{ID: missingID, SeriesID: seriesID, RelativePath: "Season 01/episode02.mkv"},
			},
		},
	}
	sr := &mockSeries{series: []SeriesInfo{{ID: seriesID, Path: dir}}}
	hist := &mockHistory{}
	vac := &mockVacuumer{}
	stats := &mockStats{}

	r := New(Options{
		History:      hist,
		EpisodeFiles: ef,
		Series:       sr,
		Stats:        stats,
		DB:           vac,
		Log:          discardLogger(),
	})
	r.cleanOrphanEpisodeFiles(context.Background())

	if len(ef.deleteCalls) != 1 {
		t.Fatalf("expected 1 delete call, got %d", len(ef.deleteCalls))
	}
	if ef.deleteCalls[0] != missingID {
		t.Errorf("expected delete of ID %d (missing), got %d", missingID, ef.deleteCalls[0])
	}
}

func TestRecalculateStatistics(t *testing.T) {
	sr := &mockSeries{series: []SeriesInfo{
		{ID: 1, Path: "/srv/media/show1"},
		{ID: 2, Path: "/srv/media/show2"},
		{ID: 3, Path: "/srv/media/show3"},
	}}
	stats := &mockStats{}
	hist := &mockHistory{}
	vac := &mockVacuumer{}
	ef := &mockEpisodeFiles{files: map[int64][]EpisodeFileInfo{}}

	r := New(Options{
		History:      hist,
		EpisodeFiles: ef,
		Series:       sr,
		Stats:        stats,
		DB:           vac,
		Log:          discardLogger(),
	})
	r.recalculateStatistics(context.Background())

	if len(stats.recomputeCalls) != 3 {
		t.Fatalf("expected Recompute called 3 times, got %d", len(stats.recomputeCalls))
	}
}

func TestVacuumDatabase(t *testing.T) {
	vac := &mockVacuumer{}
	hist := &mockHistory{}
	sr := &mockSeries{}
	stats := &mockStats{}
	ef := &mockEpisodeFiles{files: map[int64][]EpisodeFileInfo{}}

	r := New(Options{
		History:      hist,
		EpisodeFiles: ef,
		Series:       sr,
		Stats:        stats,
		DB:           vac,
		Log:          discardLogger(),
	})
	r.vacuumDatabase(context.Background())

	if !vac.called {
		t.Fatal("expected Vacuum to be called")
	}
}

func TestRunAllContinuesOnError(t *testing.T) {
	// history trimmer errors; vacuum should still run
	vac := &mockVacuumer{}
	sr := &mockSeries{}
	stats := &mockStats{}
	ef := &mockEpisodeFiles{files: map[int64][]EpisodeFileInfo{}}

	r := New(Options{
		History:      &errHistory{},
		EpisodeFiles: ef,
		Series:       sr,
		Stats:        stats,
		DB:           vac,
		Log:          discardLogger(),
	})
	r.Run(context.Background())

	if !vac.called {
		t.Fatal("expected Vacuum to be called even after history trim error")
	}
}
