package library

import (
	"context"
	"errors"
	"testing"
)

func TestStatsStoreGetReturnsErrNotFoundWhenNoRow(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	_, err := lib.Stats.Get(context.Background(), series.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestStatsStoreRecomputeEmptySeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	if err := lib.Stats.Recompute(context.Background(), series.ID); err != nil {
		t.Fatalf("Recompute: %v", err)
	}

	got, err := lib.Stats.Get(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EpisodeCount != 0 || got.EpisodeFileCount != 0 {
		t.Errorf("empty stats = %+v", got)
	}
}

func TestStatsStoreRecomputeAfterAddingEpisodes(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	// 4 episodes, 3 monitored.
	for i := int32(1); i <= 4; i++ {
		if _, err := lib.Episodes.Create(context.Background(), Episode{
			SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: i,
			Title: "E", Monitored: i <= 3,
		}); err != nil {
			t.Fatalf("Create episode: %v", err)
		}
	}

	// 2 episode files, 1500 bytes total.
	for _, size := range []int64{500, 1000} {
		if _, err := lib.EpisodeFiles.Create(context.Background(), EpisodeFile{
			SeriesID: series.ID, SeasonNumber: 1, RelativePath: "x.mkv", Size: size,
		}); err != nil {
			t.Fatalf("Create file: %v", err)
		}
	}

	if err := lib.Stats.Recompute(context.Background(), series.ID); err != nil {
		t.Fatalf("Recompute: %v", err)
	}

	got, err := lib.Stats.Get(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EpisodeCount != 4 {
		t.Errorf("EpisodeCount = %d, want 4", got.EpisodeCount)
	}
	if got.MonitoredEpisodeCount != 3 {
		t.Errorf("MonitoredEpisodeCount = %d, want 3", got.MonitoredEpisodeCount)
	}
	if got.EpisodeFileCount != 2 {
		t.Errorf("EpisodeFileCount = %d, want 2", got.EpisodeFileCount)
	}
	if got.SizeOnDisk != 1500 {
		t.Errorf("SizeOnDisk = %d, want 1500", got.SizeOnDisk)
	}
}

func TestStatsStoreRecomputeIsIdempotent(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	if err := lib.Stats.Recompute(context.Background(), series.ID); err != nil {
		t.Fatalf("first Recompute: %v", err)
	}
	if err := lib.Stats.Recompute(context.Background(), series.ID); err != nil {
		t.Fatalf("second Recompute: %v", err)
	}
	got, err := lib.Stats.Get(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EpisodeCount != 0 {
		t.Errorf("EpisodeCount = %d, want 0", got.EpisodeCount)
	}
}

func TestStatsStoreDelete(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	if err := lib.Stats.Recompute(context.Background(), series.ID); err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if err := lib.Stats.Delete(context.Background(), series.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := lib.Stats.Get(context.Background(), series.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete error = %v, want ErrNotFound", err)
	}
}
