package specs_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/blocklist"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/decisionengine"
	"github.com/ajthom90/sonarr2/internal/decisionengine/specs"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

func newBlocklistStore(t *testing.T) blocklist.Store {
	t.Helper()
	ctx := context.Background()
	pool, err := db.OpenSQLite(ctx, db.SQLiteOptions{DSN: ":memory:", BusyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("OpenSQLite: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	return blocklist.NewSQLiteStore(pool)
}

func TestBlocklistedSpecRejects(t *testing.T) {
	store := newBlocklistStore(t)
	ctx := context.Background()
	if _, err := store.Create(ctx, blocklist.Entry{
		SeriesID: 42, SourceTitle: "Show.S01E01.1080p.WEB-DL", Date: time.Now(),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	spec := specs.BlocklistedSpec{Store: store}
	remote := decisionengine.RemoteEpisode{
		SeriesID: 42,
		Release:  decisionengine.Release{Title: "Show.S01E01.1080p.WEB-DL"},
	}
	decision, rejections := spec.Evaluate(ctx, remote, profiles.QualityProfile{})
	if decision != decisionengine.Reject {
		t.Errorf("expected Reject, got %v", decision)
	}
	if len(rejections) != 1 || rejections[0].Type != decisionengine.Permanent {
		t.Errorf("expected permanent rejection, got %+v", rejections)
	}
}

func TestBlocklistedSpecAcceptsNonBlocklisted(t *testing.T) {
	store := newBlocklistStore(t)
	spec := specs.BlocklistedSpec{Store: store}
	remote := decisionengine.RemoteEpisode{
		SeriesID: 42,
		Release:  decisionengine.Release{Title: "Show.S01E01.HDTV"},
	}
	decision, _ := spec.Evaluate(context.Background(), remote, profiles.QualityProfile{})
	if decision != decisionengine.Accept {
		t.Errorf("expected Accept, got %v", decision)
	}
}

func TestBlocklistedSpecNilStore(t *testing.T) {
	spec := specs.BlocklistedSpec{Store: nil}
	remote := decisionengine.RemoteEpisode{SeriesID: 1, Release: decisionengine.Release{Title: "anything"}}
	decision, _ := spec.Evaluate(context.Background(), remote, profiles.QualityProfile{})
	if decision != decisionengine.Accept {
		t.Error("nil store should Accept")
	}
}
