package library

import (
	"context"
	"errors"
	"testing"
)

func TestEpisodeFilesStoreCreateAndGet(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	created, err := lib.EpisodeFiles.Create(context.Background(), EpisodeFile{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		RelativePath: "Season 01/Show - S01E01 - Pilot.mkv",
		Size:         1_500_000_000,
		ReleaseGroup: "ABCD",
		QualityName:  "WEBDL-1080p",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID must be non-zero")
	}
	if created.Size != 1_500_000_000 {
		t.Errorf("Size = %d, want 1500000000", created.Size)
	}

	got, err := lib.EpisodeFiles.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RelativePath != created.RelativePath {
		t.Errorf("RelativePath mismatch: %+v", got)
	}

	added := filterEvents[EpisodeFileAdded](getEvents())
	if len(added) != 1 {
		t.Errorf("EpisodeFileAdded count = %d, want 1", len(added))
	}
	if added[0].SeriesID != series.ID || added[0].Size != 1_500_000_000 {
		t.Errorf("EpisodeFileAdded = %+v", added[0])
	}
}

func TestEpisodeFilesStoreGetMissing(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	_, err := lib.EpisodeFiles.Get(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestEpisodeFilesStoreListForSeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	for i, path := range []string{"b.mkv", "a.mkv", "c.mkv"} {
		if _, err := lib.EpisodeFiles.Create(context.Background(), EpisodeFile{
			SeriesID: series.ID, SeasonNumber: int32(i + 1), RelativePath: path, Size: 100,
		}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := lib.EpisodeFiles.ListForSeries(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
}

func TestEpisodeFilesStoreDelete(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	created, err := lib.EpisodeFiles.Create(context.Background(), EpisodeFile{
		SeriesID: series.ID, SeasonNumber: 1, RelativePath: "x.mkv", Size: 100,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := lib.EpisodeFiles.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = lib.EpisodeFiles.Get(context.Background(), created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}

	deleted := filterEvents[EpisodeFileDeleted](getEvents())
	if len(deleted) != 1 {
		t.Errorf("EpisodeFileDeleted count = %d, want 1", len(deleted))
	}
	if deleted[0].SeriesID != series.ID {
		t.Errorf("EpisodeFileDeleted.SeriesID = %d, want %d", deleted[0].SeriesID, series.ID)
	}
}

func TestEpisodeFilesStoreSumSizesForSeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	for _, size := range []int64{100, 200, 300} {
		if _, err := lib.EpisodeFiles.Create(context.Background(), EpisodeFile{
			SeriesID: series.ID, SeasonNumber: 1, RelativePath: "x.mkv", Size: size,
		}); err != nil {
			// Each Create must have a unique relative_path... but our schema
			// doesn't enforce uniqueness on relative_path. Re-check: the
			// schema only has a series_id+season_number index, not unique.
			// So duplicate paths are allowed — this test stays simple.
			t.Fatalf("Create: %v", err)
		}
	}

	count, size, err := lib.EpisodeFiles.SumSizesForSeries(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("SumSizesForSeries: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if size != 600 {
		t.Errorf("size = %d, want 600", size)
	}
}
