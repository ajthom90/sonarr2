package library

import (
	"context"
	"errors"
	"testing"
	"time"
)

func createSeries(t *testing.T, lib *Library, tvdbID int64) Series {
	t.Helper()
	s, err := lib.Series.Create(context.Background(), Series{
		TvdbID: tvdbID, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create series: %v", err)
	}
	return s
}

func TestEpisodesStoreCreateAndGet(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	airDate := time.Date(2026, 4, 11, 0, 0, 0, 0, time.UTC)
	absNum := int32(1)
	created, err := lib.Episodes.Create(context.Background(), Episode{
		SeriesID:              series.ID,
		SeasonNumber:          1,
		EpisodeNumber:         1,
		AbsoluteEpisodeNumber: &absNum,
		Title:                 "Pilot",
		Overview:              "First episode",
		AirDateUtc:            &airDate,
		Monitored:             true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID must be non-zero")
	}
	if created.Title != "Pilot" {
		t.Errorf("Title = %q, want Pilot", created.Title)
	}
	if created.AbsoluteEpisodeNumber == nil || *created.AbsoluteEpisodeNumber != 1 {
		t.Errorf("AbsoluteEpisodeNumber = %v, want *1", created.AbsoluteEpisodeNumber)
	}

	got, err := lib.Episodes.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Pilot" || got.SeriesID != series.ID {
		t.Errorf("roundtrip mismatch: %+v", got)
	}

	added := filterEvents[EpisodeAdded](getEvents())
	if len(added) != 1 {
		t.Errorf("EpisodeAdded count = %d, want 1", len(added))
	}
	if added[0].SeriesID != series.ID {
		t.Errorf("EpisodeAdded.SeriesID = %d, want %d", added[0].SeriesID, series.ID)
	}
}

func TestEpisodesStoreGetReturnsErrNotFound(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	_, err := lib.Episodes.Get(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get error = %v, want ErrNotFound", err)
	}
}

func TestEpisodesStoreListForSeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	for _, n := range []int32{3, 1, 2} {
		if _, err := lib.Episodes.Create(context.Background(), Episode{
			SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: n,
			Title: "E", Monitored: true,
		}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := lib.Episodes.ListForSeries(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
	if list[0].EpisodeNumber != 1 || list[2].EpisodeNumber != 3 {
		t.Errorf("order not sorted: %+v", list)
	}
}

func TestEpisodesStoreUpdate(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	ep, err := lib.Episodes.Create(context.Background(), Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 1,
		Title: "Original", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ep.Title = "Renamed"
	ep.Monitored = false
	if err := lib.Episodes.Update(context.Background(), ep); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := lib.Episodes.Get(context.Background(), ep.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Renamed" || got.Monitored {
		t.Errorf("after update: %+v", got)
	}

	updated := filterEvents[EpisodeUpdated](getEvents())
	if len(updated) != 1 {
		t.Errorf("EpisodeUpdated count = %d, want 1", len(updated))
	}
}

func TestEpisodesStoreDelete(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	ep, err := lib.Episodes.Create(context.Background(), Episode{
		SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: 1,
		Title: "E", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := lib.Episodes.Delete(context.Background(), ep.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = lib.Episodes.Get(context.Background(), ep.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}

	deleted := filterEvents[EpisodeDeleted](getEvents())
	if len(deleted) != 1 {
		t.Errorf("EpisodeDeleted count = %d, want 1", len(deleted))
	}
	if deleted[0].SeriesID != series.ID {
		t.Errorf("EpisodeDeleted.SeriesID = %d, want %d", deleted[0].SeriesID, series.ID)
	}
}

func TestEpisodesStoreCountForSeries(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)
	series := createSeries(t, lib, 1)

	for i := int32(1); i <= 5; i++ {
		monitored := i%2 == 1 // 3 of 5 monitored
		if _, err := lib.Episodes.Create(context.Background(), Episode{
			SeriesID: series.ID, SeasonNumber: 1, EpisodeNumber: i,
			Title: "E", Monitored: monitored,
		}); err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	total, monitored, err := lib.Episodes.CountForSeries(context.Background(), series.ID)
	if err != nil {
		t.Fatalf("CountForSeries: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if monitored != 3 {
		t.Errorf("monitored = %d, want 3", monitored)
	}
}
