package rootfolder_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/rootfolder"
)

func newTestStore(t *testing.T) rootfolder.Store {
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
	return rootfolder.NewSQLiteStore(pool)
}

func TestStore_CreateGetList(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, "/tv")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero id")
	}
	if created.Path != "/tv" {
		t.Errorf("path = %q, want %q", created.Path, "/tv")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	got, err := store.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Path != "/tv" {
		t.Errorf("got path %q", got.Path)
	}

	byPath, err := store.GetByPath(ctx, "/tv")
	if err != nil {
		t.Fatalf("GetByPath: %v", err)
	}
	if byPath.ID != created.ID {
		t.Errorf("byPath id = %d, want %d", byPath.ID, created.ID)
	}

	// Add a second folder so List has more than one entry.
	if _, err := store.Create(ctx, "/anime"); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 root folders, got %d", len(list))
	}
	// ListRootFolders orders by path, so "/anime" should come before "/tv".
	if list[0].Path != "/anime" || list[1].Path != "/tv" {
		t.Errorf("list order = [%q, %q], want [%q, %q]",
			list[0].Path, list[1].Path, "/anime", "/tv")
	}
}

func TestStore_CreateDuplicate(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "/tv"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := store.Create(ctx, "/tv")
	if !errors.Is(err, rootfolder.ErrAlreadyExists) {
		t.Errorf("want ErrAlreadyExists, got %v", err)
	}
}

func TestStore_Delete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, "/tv")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, created.ID); !errors.Is(err, rootfolder.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestStore_GetByPathNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetByPath(context.Background(), "/does/not/exist")
	if !errors.Is(err, rootfolder.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}
