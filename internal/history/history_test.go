package history_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/history"
)

func newTestStore(t *testing.T) history.Store {
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
	return history.NewSQLiteStore(pool)
}

func TestHistoryCreateAndListForSeries(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	e1 := history.Entry{
		EpisodeID:   1,
		SeriesID:    10,
		SourceTitle: "Show.S01E01.720p",
		QualityName: "720p",
		EventType:   history.EventGrabbed,
		DownloadID:  "abc123",
		Data:        json.RawMessage(`{"indexer":"NZBGeek"}`),
	}
	e2 := history.Entry{
		EpisodeID:   2,
		SeriesID:    10,
		SourceTitle: "Show.S01E02.1080p",
		QualityName: "1080p",
		EventType:   history.EventDownloadImported,
		DownloadID:  "def456",
		Data:        json.RawMessage(`{"indexer":"NZBGeek"}`),
	}

	created1, err := store.Create(ctx, e1)
	if err != nil {
		t.Fatalf("Create e1: %v", err)
	}
	if created1.ID == 0 {
		t.Error("created1.ID must be non-zero")
	}

	// Small sleep to ensure second entry has a later timestamp.
	time.Sleep(2 * time.Millisecond)

	created2, err := store.Create(ctx, e2)
	if err != nil {
		t.Fatalf("Create e2: %v", err)
	}
	if created2.ID == 0 {
		t.Error("created2.ID must be non-zero")
	}
	if created2.ID == created1.ID {
		t.Error("created2.ID must differ from created1.ID")
	}

	list, err := store.ListForSeries(ctx, 10)
	if err != nil {
		t.Fatalf("ListForSeries: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("ListForSeries len = %d, want 2", len(list))
	}

	// Verify fields on the first returned entry.
	if list[0].SeriesID != 10 {
		t.Errorf("list[0].SeriesID = %d, want 10", list[0].SeriesID)
	}
	if list[0].SourceTitle == "" {
		t.Error("list[0].SourceTitle must not be empty")
	}

	// ListForSeries for a different series returns nothing.
	other, err := store.ListForSeries(ctx, 99)
	if err != nil {
		t.Fatalf("ListForSeries(99): %v", err)
	}
	if len(other) != 0 {
		t.Errorf("ListForSeries(99) len = %d, want 0", len(other))
	}
}

func TestHistoryListForEpisode(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Two entries for different episodes under the same series.
	_, err := store.Create(ctx, history.Entry{
		EpisodeID:   100,
		SeriesID:    20,
		SourceTitle: "Show.S02E01",
		EventType:   history.EventGrabbed,
	})
	if err != nil {
		t.Fatalf("Create ep100: %v", err)
	}

	_, err = store.Create(ctx, history.Entry{
		EpisodeID:   101,
		SeriesID:    20,
		SourceTitle: "Show.S02E02",
		EventType:   history.EventGrabbed,
	})
	if err != nil {
		t.Fatalf("Create ep101: %v", err)
	}

	// ListForEpisode should return only the matching episode.
	list, err := store.ListForEpisode(ctx, 100)
	if err != nil {
		t.Fatalf("ListForEpisode(100): %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListForEpisode(100) len = %d, want 1", len(list))
	}
	if list[0].EpisodeID != 100 {
		t.Errorf("list[0].EpisodeID = %d, want 100", list[0].EpisodeID)
	}
	if list[0].SourceTitle != "Show.S02E01" {
		t.Errorf("list[0].SourceTitle = %q, want Show.S02E01", list[0].SourceTitle)
	}

	// The other episode returns only its own entry.
	list2, err := store.ListForEpisode(ctx, 101)
	if err != nil {
		t.Fatalf("ListForEpisode(101): %v", err)
	}
	if len(list2) != 1 {
		t.Fatalf("ListForEpisode(101) len = %d, want 1", len(list2))
	}
	if list2[0].EpisodeID != 101 {
		t.Errorf("list2[0].EpisodeID = %d, want 101", list2[0].EpisodeID)
	}
}

func TestHistoryFindByDownloadID(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_, err := store.Create(ctx, history.Entry{
		EpisodeID:   200,
		SeriesID:    30,
		SourceTitle: "Show.S03E01",
		EventType:   history.EventGrabbed,
		DownloadID:  "unique-dl-id",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Another entry with a different download_id.
	_, err = store.Create(ctx, history.Entry{
		EpisodeID:   201,
		SeriesID:    30,
		SourceTitle: "Show.S03E02",
		EventType:   history.EventGrabbed,
		DownloadID:  "other-dl-id",
	})
	if err != nil {
		t.Fatalf("Create other: %v", err)
	}

	found, err := store.FindByDownloadID(ctx, "unique-dl-id")
	if err != nil {
		t.Fatalf("FindByDownloadID: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("FindByDownloadID len = %d, want 1", len(found))
	}
	if found[0].DownloadID != "unique-dl-id" {
		t.Errorf("found[0].DownloadID = %q, want unique-dl-id", found[0].DownloadID)
	}
	if found[0].EpisodeID != 200 {
		t.Errorf("found[0].EpisodeID = %d, want 200", found[0].EpisodeID)
	}

	// Looking up a nonexistent download_id returns empty slice.
	none, err := store.FindByDownloadID(ctx, "no-such-id")
	if err != nil {
		t.Fatalf("FindByDownloadID(missing): %v", err)
	}
	if len(none) != 0 {
		t.Errorf("FindByDownloadID(missing) len = %d, want 0", len(none))
	}
}

func TestHistoryDeleteForSeries(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	// Create two entries for series 40 and one for series 41.
	for _, ep := range []int64{300, 301} {
		_, err := store.Create(ctx, history.Entry{
			EpisodeID:   ep,
			SeriesID:    40,
			SourceTitle: "Show.S04E0x",
			EventType:   history.EventGrabbed,
		})
		if err != nil {
			t.Fatalf("Create ep%d: %v", ep, err)
		}
	}
	_, err := store.Create(ctx, history.Entry{
		EpisodeID:   400,
		SeriesID:    41,
		SourceTitle: "Other.S01E01",
		EventType:   history.EventGrabbed,
	})
	if err != nil {
		t.Fatalf("Create other series: %v", err)
	}

	// Confirm two entries for series 40 before deletion.
	before, err := store.ListForSeries(ctx, 40)
	if err != nil {
		t.Fatalf("ListForSeries before delete: %v", err)
	}
	if len(before) != 2 {
		t.Fatalf("before delete len = %d, want 2", len(before))
	}

	// Delete all history for series 40.
	if err := store.DeleteForSeries(ctx, 40); err != nil {
		t.Fatalf("DeleteForSeries: %v", err)
	}

	// Series 40 entries are gone.
	after, err := store.ListForSeries(ctx, 40)
	if err != nil {
		t.Fatalf("ListForSeries after delete: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("after delete len = %d, want 0", len(after))
	}

	// Series 41 entry is unaffected.
	other, err := store.ListForSeries(ctx, 41)
	if err != nil {
		t.Fatalf("ListForSeries series 41: %v", err)
	}
	if len(other) != 1 {
		t.Errorf("series 41 after delete len = %d, want 1", len(other))
	}
}
