package profiles_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

func newTestPool(t *testing.T) *db.SQLitePool {
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
	return pool
}

// TestQualityDefinitionsSeeded verifies that the 18 standard quality
// definitions inserted by migration 00009 are all present and readable.
func TestQualityDefinitionsSeeded(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityDefinitionStore(pool)

	defs, err := store.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(defs) != 18 {
		t.Errorf("want 18 seeded quality definitions, got %d", len(defs))
	}

	// Quick sanity check: first definition should be SDTV.
	if defs[0].Name != "SDTV" {
		t.Errorf("defs[0].Name = %q, want SDTV", defs[0].Name)
	}
}

// TestQualityDefinitionGetByID verifies that GetByID returns a correct row
// and ErrNotFound for a missing ID.
func TestQualityDefinitionGetByID(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityDefinitionStore(pool)

	all, err := store.GetAll(context.Background())
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("no definitions to test with")
	}

	first := all[0]
	got, err := store.GetByID(context.Background(), first.ID)
	if err != nil {
		t.Fatalf("GetByID(%d): %v", first.ID, err)
	}
	if got.Name != first.Name {
		t.Errorf("Name = %q, want %q", got.Name, first.Name)
	}

	_, err = store.GetByID(context.Background(), 9999)
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

// TestQualityProfileCRUD tests Create/Get/List/Update/Delete roundtrip.
func TestQualityProfileCRUD(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityProfileStore(pool)
	ctx := context.Background()

	in := profiles.QualityProfile{
		Name:              "Test Profile",
		UpgradeAllowed:    true,
		Cutoff:            10,
		Items:             []profiles.QualityProfileItem{{QualityID: 1, Allowed: true}, {QualityID: 2, Allowed: false}},
		MinFormatScore:    0,
		CutoffFormatScore: 50,
		FormatItems:       []profiles.FormatScoreItem{{FormatID: 5, Score: 10}},
	}

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

	// Get
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, in.Name)
	}
	if got.UpgradeAllowed != in.UpgradeAllowed {
		t.Errorf("UpgradeAllowed = %v, want %v", got.UpgradeAllowed, in.UpgradeAllowed)
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
	got.Name = "Renamed Profile"
	got.UpgradeAllowed = false
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.GetByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.Name != "Renamed Profile" {
		t.Errorf("Name after update = %q, want Renamed Profile", after.Name)
	}
	if after.UpgradeAllowed {
		t.Error("UpgradeAllowed after update = true, want false")
	}

	// Delete
	if err := store.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.GetByID(ctx, got.ID)
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

// TestQualityProfileGetByIDNotFound verifies ErrNotFound for missing IDs.
func TestQualityProfileGetByIDNotFound(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityProfileStore(pool)

	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, profiles.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

// TestQualityProfileItemsJSONRoundtrip verifies that Items and FormatItems
// survive a create/get cycle with correct values.
func TestQualityProfileItemsJSONRoundtrip(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityProfileStore(pool)
	ctx := context.Background()

	items := []profiles.QualityProfileItem{
		{QualityID: 10, Allowed: true},
		{QualityID: 11, Allowed: false},
		{QualityID: 12, Allowed: true},
	}
	formatItems := []profiles.FormatScoreItem{
		{FormatID: 1, Score: 25},
		{FormatID: 2, Score: -10},
	}

	created, err := store.Create(ctx, profiles.QualityProfile{
		Name:        "JSON Test",
		Items:       items,
		FormatItems: formatItems,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if len(got.Items) != len(items) {
		t.Fatalf("Items len = %d, want %d", len(got.Items), len(items))
	}
	for i, want := range items {
		if got.Items[i] != want {
			t.Errorf("Items[%d] = %+v, want %+v", i, got.Items[i], want)
		}
	}

	if len(got.FormatItems) != len(formatItems) {
		t.Fatalf("FormatItems len = %d, want %d", len(got.FormatItems), len(formatItems))
	}
	for i, want := range formatItems {
		if got.FormatItems[i] != want {
			t.Errorf("FormatItems[%d] = %+v, want %+v", i, got.FormatItems[i], want)
		}
	}
}

// TestQualityProfileEmptyItemsRoundtrip verifies that nil/empty slices work.
func TestQualityProfileEmptyItemsRoundtrip(t *testing.T) {
	pool := newTestPool(t)
	store := profiles.NewSQLiteQualityProfileStore(pool)
	ctx := context.Background()

	created, err := store.Create(ctx, profiles.QualityProfile{
		Name:        "Empty Items",
		Items:       []profiles.QualityProfileItem{},
		FormatItems: []profiles.FormatScoreItem{},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Items == nil {
		t.Error("Items should be non-nil empty slice")
	}
	if len(got.Items) != 0 {
		t.Errorf("Items len = %d, want 0", len(got.Items))
	}
	if got.FormatItems == nil {
		t.Error("FormatItems should be non-nil empty slice")
	}
	if len(got.FormatItems) != 0 {
		t.Errorf("FormatItems len = %d, want 0", len(got.FormatItems))
	}
}
