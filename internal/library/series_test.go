package library

import (
	"context"
	"errors"
	"testing"
)

func TestSeriesStoreCreateAndGet(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)

	want := Series{
		TvdbID:     12345,
		Title:      "Test Show",
		Slug:       "test-show",
		Status:     "continuing",
		SeriesType: "standard",
		Path:       "/tv/Test Show",
		Monitored:  true,
	}

	created, err := lib.Series.Create(context.Background(), want)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID must be non-zero")
	}
	if created.Title != want.Title {
		t.Errorf("Title = %q, want %q", created.Title, want.Title)
	}

	got, err := lib.Series.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID || got.Title != want.Title {
		t.Errorf("roundtrip mismatch: got %+v, want (ID=%d title=%q)", got, created.ID, want.Title)
	}

	added := assertHasEvent[SeriesAdded](t, getEvents)
	if added.ID != created.ID {
		t.Errorf("SeriesAdded.ID = %d, want %d", added.ID, created.ID)
	}
	if added.Title != want.Title {
		t.Errorf("SeriesAdded.Title = %q, want %q", added.Title, want.Title)
	}
}

func TestSeriesStoreGetReturnsErrNotFoundWhenMissing(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	_, err := lib.Series.Get(context.Background(), 999)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(missing) error = %v, want ErrNotFound", err)
	}
}

func TestSeriesStoreGetByTvdbID(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	_, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 99, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := lib.Series.GetByTvdbID(context.Background(), 99)
	if err != nil {
		t.Fatalf("GetByTvdbID: %v", err)
	}
	if got.TvdbID != 99 {
		t.Errorf("TvdbID = %d, want 99", got.TvdbID)
	}

	_, err = lib.Series.GetByTvdbID(context.Background(), 12345)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("missing GetByTvdbID error = %v, want ErrNotFound", err)
	}
}

func TestSeriesStoreGetBySlug(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	_, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 1, Title: "A", Slug: "a", Path: "/tv/A",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := lib.Series.GetBySlug(context.Background(), "a")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if got.Slug != "a" {
		t.Errorf("Slug = %q, want a", got.Slug)
	}
}

func TestSeriesStoreList(t *testing.T) {
	lib, _, _, _ := setupSQLiteLibrary(t)

	for i, slug := range []string{"b-show", "a-show", "c-show"} {
		_, err := lib.Series.Create(context.Background(), Series{
			TvdbID: int64(i + 1), Title: slug, Slug: slug, Path: "/tv/" + slug,
			Status: "continuing", SeriesType: "standard", Monitored: true,
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	list, err := lib.Series.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("len = %d, want 3", len(list))
	}
	// Ordered by title — "a-show" first.
	if list[0].Slug != "a-show" {
		t.Errorf("list[0].Slug = %q, want a-show", list[0].Slug)
	}
}

func TestSeriesStoreUpdate(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)

	created, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 100, Title: "Original", Slug: "original", Path: "/tv/Original",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	created.Title = "Renamed"
	created.Monitored = false
	if err := lib.Series.Update(context.Background(), created); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := lib.Series.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Title != "Renamed" {
		t.Errorf("Title = %q, want Renamed", got.Title)
	}
	if got.Monitored {
		t.Error("Monitored = true, want false")
	}

	updated := filterEvents[SeriesUpdated](getEvents())
	if len(updated) != 1 {
		t.Errorf("SeriesUpdated count = %d, want 1", len(updated))
	}
}

func TestSeriesStoreDelete(t *testing.T) {
	lib, _, _, getEvents := setupSQLiteLibrary(t)

	created, err := lib.Series.Create(context.Background(), Series{
		TvdbID: 200, Title: "X", Slug: "x", Path: "/tv/X",
		Status: "continuing", SeriesType: "standard", Monitored: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := lib.Series.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err = lib.Series.Get(context.Background(), created.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Get after delete error = %v, want ErrNotFound", err)
	}

	deleted := filterEvents[SeriesDeleted](getEvents())
	if len(deleted) != 1 {
		t.Errorf("SeriesDeleted count = %d, want 1", len(deleted))
	}
	if deleted[0].ID != created.ID {
		t.Errorf("SeriesDeleted.ID = %d, want %d", deleted[0].ID, created.ID)
	}
}
