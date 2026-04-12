package indexer_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

func newTestIndexerStore(t *testing.T) indexer.InstanceStore {
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
	return indexer.NewSQLiteInstanceStore(pool)
}

func TestIndexerStoreCRUD(t *testing.T) {
	store := newTestIndexerStore(t)
	ctx := context.Background()

	in := indexer.Instance{
		Name:                    "My Indexer",
		Implementation:          "Newznab",
		Settings:                json.RawMessage(`{"baseUrl":"https://example.com","apiKey":"secret"}`),
		EnableRss:               true,
		EnableAutomaticSearch:   true,
		EnableInteractiveSearch: false,
		Priority:                25,
	}

	// Create
	created, err := store.Create(ctx, in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("created.ID must be non-zero")
	}
	if created.Name != in.Name {
		t.Errorf("Name = %q, want %q", created.Name, in.Name)
	}
	if created.Implementation != in.Implementation {
		t.Errorf("Implementation = %q, want %q", created.Implementation, in.Implementation)
	}
	if created.EnableRss != in.EnableRss {
		t.Errorf("EnableRss = %v, want %v", created.EnableRss, in.EnableRss)
	}
	if created.EnableAutomaticSearch != in.EnableAutomaticSearch {
		t.Errorf("EnableAutomaticSearch = %v, want %v", created.EnableAutomaticSearch, in.EnableAutomaticSearch)
	}
	if created.EnableInteractiveSearch != in.EnableInteractiveSearch {
		t.Errorf("EnableInteractiveSearch = %v, want %v", created.EnableInteractiveSearch, in.EnableInteractiveSearch)
	}
	if created.Priority != in.Priority {
		t.Errorf("Priority = %d, want %d", created.Priority, in.Priority)
	}

	// GetByID
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, in.Name)
	}
	if got.Priority != in.Priority {
		t.Errorf("got.Priority = %d, want %d", got.Priority, in.Priority)
	}

	// List
	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("List len = %d, want 1", len(list))
	}

	// Update
	got.Name = "Updated Indexer"
	got.EnableRss = false
	got.Priority = 10
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.GetByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.Name != "Updated Indexer" {
		t.Errorf("Name after update = %q, want %q", after.Name, "Updated Indexer")
	}
	if after.EnableRss {
		t.Error("EnableRss after update = true, want false")
	}
	if after.Priority != 10 {
		t.Errorf("Priority after update = %d, want 10", after.Priority)
	}

	// Delete
	if err := store.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.GetByID(ctx, got.ID)
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

func TestIndexerStoreGetByIDNotFound(t *testing.T) {
	store := newTestIndexerStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, indexer.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

func TestIndexerStoreSettingsJSONRoundtrip(t *testing.T) {
	store := newTestIndexerStore(t)
	ctx := context.Background()

	settings := json.RawMessage(`{"baseUrl":"https://nzb.example.com","apiPath":"/api","apiKey":"abc123","categories":[5030,5040]}`)

	created, err := store.Create(ctx, indexer.Instance{
		Name:           "Roundtrip Indexer",
		Implementation: "Newznab",
		Settings:       settings,
		Priority:       25,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	// Normalize both JSON values for comparison.
	var wantNorm, gotNorm any
	if err := json.Unmarshal(settings, &wantNorm); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if err := json.Unmarshal(got.Settings, &gotNorm); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	wantBytes, _ := json.Marshal(wantNorm)
	gotBytes, _ := json.Marshal(gotNorm)
	if !bytes.Equal(wantBytes, gotBytes) {
		t.Errorf("Settings roundtrip mismatch:\n  want %s\n   got %s", wantBytes, gotBytes)
	}
}
