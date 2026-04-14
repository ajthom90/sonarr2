package blocklist_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/blocklist"
	"github.com/ajthom90/sonarr2/internal/db"
)

func newTestStore(t *testing.T) blocklist.Store {
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
	return blocklist.NewSQLiteStore(pool)
}

func sampleEntry(seriesID int, title string) blocklist.Entry {
	q, _ := json.Marshal(map[string]any{"quality": map[string]int{"id": 5}})
	l, _ := json.Marshal([]map[string]any{{"id": 1, "name": "English"}})
	return blocklist.Entry{
		SeriesID:     seriesID,
		EpisodeIDs:   []int{10, 11},
		SourceTitle:  title,
		Quality:      q,
		Languages:    l,
		Date:         time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC),
		Protocol:     blocklist.ProtocolTorrent,
		Indexer:      "Test Indexer",
		IndexerFlags: 0,
		Message:      "manual blocklist",
	}
}

func TestCreateAndGet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, sampleEntry(1, "Show.S01E01.1080p.WEB-DL"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero id")
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SourceTitle != "Show.S01E01.1080p.WEB-DL" {
		t.Errorf("source title mismatch: %q", got.SourceTitle)
	}
	if len(got.EpisodeIDs) != 2 || got.EpisodeIDs[0] != 10 {
		t.Errorf("episodeIds mismatch: %v", got.EpisodeIDs)
	}
	if got.Protocol != blocklist.ProtocolTorrent {
		t.Errorf("protocol mismatch: %q", got.Protocol)
	}
}

func TestListPagination(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := store.Create(ctx, sampleEntry(1, "title"+string(rune('A'+i)))); err != nil {
			t.Fatal(err)
		}
	}
	pg, err := store.List(ctx, 1, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if pg.TotalRecords != 5 {
		t.Errorf("total = %d, want 5", pg.TotalRecords)
	}
	if len(pg.Records) != 3 {
		t.Errorf("page size = %d, want 3", len(pg.Records))
	}
	pg2, err := store.List(ctx, 2, 3)
	if err != nil {
		t.Fatalf("List page 2: %v", err)
	}
	if len(pg2.Records) != 2 {
		t.Errorf("page 2 size = %d, want 2", len(pg2.Records))
	}
}

func TestDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	created, _ := store.Create(ctx, sampleEntry(1, "x"))
	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.GetByID(ctx, created.ID); !errors.Is(err, blocklist.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestDeleteMany(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()
	var ids []int
	for i := 0; i < 3; i++ {
		e, _ := store.Create(ctx, sampleEntry(1, "t"+string(rune('a'+i))))
		ids = append(ids, e.ID)
	}
	if err := store.DeleteMany(ctx, ids); err != nil {
		t.Fatalf("DeleteMany: %v", err)
	}
	pg, _ := store.List(ctx, 1, 100)
	if pg.TotalRecords != 0 {
		t.Errorf("expected 0 rows after DeleteMany, got %d", pg.TotalRecords)
	}
}

func TestMatches(t *testing.T) {
	entries := []blocklist.Entry{
		{SeriesID: 1, SourceTitle: "A"},
		{SeriesID: 2, SourceTitle: "B"},
	}
	if !blocklist.Matches(entries, 1, "A") {
		t.Error("want match")
	}
	if blocklist.Matches(entries, 1, "B") {
		t.Error("want no match (different series)")
	}
}
