package tags_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/tags"
)

func newTestStore(t *testing.T) tags.Store {
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
	return tags.NewSQLiteStore(pool)
}

func TestCreateListGetDelete(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, "Anime")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero id")
	}
	if created.Label != "anime" {
		t.Errorf("label = %q, want normalized %q", created.Label, "anime")
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Label != "anime" {
		t.Errorf("got label %q", got.Label)
	}

	byLabel, err := store.GetByLabel(ctx, "anime")
	if err != nil {
		t.Fatalf("GetByLabel: %v", err)
	}
	if byLabel.ID != created.ID {
		t.Errorf("byLabel id = %d, want %d", byLabel.ID, created.ID)
	}

	list, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 tag, got %d", len(list))
	}

	if err := store.Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.GetByID(ctx, created.ID); !errors.Is(err, tags.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDuplicateLabelRejected(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if _, err := store.Create(ctx, "4k"); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, err := store.Create(ctx, "4K") // case-insensitive collision
	if !errors.Is(err, tags.ErrDuplicateLabel) {
		t.Errorf("want ErrDuplicateLabel, got %v", err)
	}
}

func TestUpdate(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	created, err := store.Create(ctx, "old")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Update(ctx, tags.Tag{ID: created.ID, Label: "NEW"}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Label != "new" {
		t.Errorf("label = %q, want normalized %q", got.Label, "new")
	}
}

func TestGetByIDNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, tags.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestEmptyLabelRejected(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.Create(context.Background(), "   "); err == nil {
		t.Error("expected error for empty label, got nil")
	}
}

func TestNormalizeLabel(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Anime", "anime"},
		{"  4K  ", "4k"},
		{"TV", "tv"},
	}
	for _, c := range cases {
		if got := tags.NormalizeLabel(c.in); got != c.want {
			t.Errorf("NormalizeLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
