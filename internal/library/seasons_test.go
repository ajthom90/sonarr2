package library

import (
	"context"
	"errors"
	"testing"
)

func TestSeasonsStoreUpsertAndGet(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)

	series, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 1, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	// Upsert a season.
	err = lib.Seasons.Upsert(context.Background(), Season{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    true,
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := lib.Seasons.Get(context.Background(), series.ID, 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SeriesID != series.ID || got.SeasonNumber != 1 || !got.Monitored {
		t.Errorf("got %+v", got)
	}

	// Upsert the same (series, season) with a different monitored flag.
	err = lib.Seasons.Upsert(context.Background(), Season{
		SeriesID:     series.ID,
		SeasonNumber: 1,
		Monitored:    false,
	})
	if err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	got, err = lib.Seasons.Get(context.Background(), series.ID, 1)
	if err != nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	if got.Monitored {
		t.Error("Monitored = true, want false after second upsert")
	}

	// Both upserts should have published SeasonUpdated events.
	updates := filterEvents[SeasonUpdated](getEvents())
	if len(updates) != 2 {
		t.Errorf("SeasonUpdated count = %d, want 2", len(updates))
	}
}

func TestSeasonsStoreGetMissing(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	_, err := lib.Seasons.Get(context.Background(), 1, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}

func TestSeasonsStoreListForSeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	series, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 1, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}

	for _, sn := range []int32{3, 1, 2} {
		if err := lib.Seasons.Upsert(context.Background(), Season{
			SeriesID: series.ID, SeasonNumber: sn, Monitored: true,
		}); err != nil {
			t.Fatalf("Upsert: %v", err)
		}
	}

	list, err := lib.Seasons.ListForSeries(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
	if list[0].SeasonNumber != 1 || list[2].SeasonNumber != 3 {
		t.Errorf("order = [%d, %d, %d], want [1, 2, 3]",
			list[0].SeasonNumber, list[1].SeasonNumber, list[2].SeasonNumber)
	}
}

func TestSeasonsStoreDelete(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	series, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 1, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}
	if err := lib.Seasons.Upsert(context.Background(), Season{
		SeriesID: series.ID, SeasonNumber: 1, Monitored: true,
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := lib.Seasons.Delete(context.Background(), series.ID, 1); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = lib.Seasons.Get(context.Background(), series.ID, 1)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after Delete error = %v, want ErrNotFound", err)
	}
}
