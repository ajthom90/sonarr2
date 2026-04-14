package rootfolder_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

func TestBackfillFromSeries(t *testing.T) {
	pool := newTestPool(t)
	rfStore := rootfolder.NewSQLiteStore(pool)
	seriesStore := newTestSeriesStore(t, pool)

	ctx := context.Background()

	if err := seedQualityProfile(t, pool); err != nil {
		t.Fatalf("seed quality profile: %v", err)
	}

	for i, p := range []string{
		"/data/tv/Breaking Bad",
		"/data/tv/Better Call Saul",
		"/data/anime/Spy x Family",
	} {
		if _, err := seriesStore.Create(ctx, library.Series{
			TvdbID: int64(1000 + i), Title: "Show", Slug: fmt.Sprintf("show-%d", i),
			Status: "continuing", SeriesType: "standard",
			Path: p, Monitored: true, QualityProfileID: 1, SeasonFolder: true, MonitorNewItems: "all",
		}); err != nil {
			t.Fatalf("seed series %d: %v", i, err)
		}
	}

	if err := rootfolder.BackfillFromSeries(ctx, rfStore, seriesStore); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	got, err := rfStore.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	paths := map[string]bool{}
	for _, r := range got {
		paths[r.Path] = true
	}
	wantPaths := []string{"/data/tv", "/data/anime"}
	for _, p := range wantPaths {
		if !paths[p] {
			t.Errorf("missing root folder %q (got %v)", p, paths)
		}
	}

	// Idempotency: rerunning shouldn't duplicate.
	if err := rootfolder.BackfillFromSeries(ctx, rfStore, seriesStore); err != nil {
		t.Fatalf("re-Backfill: %v", err)
	}
	got2, _ := rfStore.List(ctx)
	if len(got2) != len(got) {
		t.Errorf("backfill not idempotent: %d -> %d rows", len(got), len(got2))
	}
}

func TestBackfillFromSeries_Empty(t *testing.T) {
	pool := newTestPool(t)
	rfStore := rootfolder.NewSQLiteStore(pool)
	seriesStore := newTestSeriesStore(t, pool)

	if err := rootfolder.BackfillFromSeries(context.Background(), rfStore, seriesStore); err != nil {
		t.Fatalf("Backfill with empty series: %v", err)
	}
	rows, err := rfStore.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected 0 root folders; got %d", len(rows))
	}
}
