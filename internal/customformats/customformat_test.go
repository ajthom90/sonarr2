package customformats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/customformats"
	"github.com/ajthom90/sonarr2/internal/db"
	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// --- Matcher tests ---

func TestMatchReleaseTitleSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "REMUX",
		Specifications: []customformats.Specification{
			{
				Implementation: "ReleaseTitleSpecification",
				Value:          `(?i)\bREMUX\b`,
			},
		},
	}

	matching := parser.ParsedEpisodeInfo{ReleaseTitle: "Show.S01E01.1080p.REMUX.BluRay"}
	if !customformats.Match(matching, cf) {
		t.Error("expected REMUX release to match ReleaseTitleSpecification")
	}

	nonMatching := parser.ParsedEpisodeInfo{ReleaseTitle: "Show.S01E01.1080p.WEB-DL"}
	if customformats.Match(nonMatching, cf) {
		t.Error("expected WEB-DL release not to match REMUX specification")
	}
}

func TestMatchNegatedSpec(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "Not HDTV",
		Specifications: []customformats.Specification{
			{
				Implementation: "ReleaseTitleSpecification",
				Value:          `(?i)\bHDTV\b`,
				Negate:         true,
			},
		},
	}

	// HDTV release should NOT match "Not HDTV" format (negated spec inverts)
	hdtvRelease := parser.ParsedEpisodeInfo{ReleaseTitle: "Show.S01E01.720p.HDTV.x264"}
	if customformats.Match(hdtvRelease, cf) {
		t.Error("HDTV release should not match negated HDTV spec")
	}

	// Non-HDTV release SHOULD match "Not HDTV" format
	webRelease := parser.ParsedEpisodeInfo{ReleaseTitle: "Show.S01E01.720p.WEB-DL"}
	if !customformats.Match(webRelease, cf) {
		t.Error("non-HDTV release should match negated HDTV spec")
	}
}

func TestMatchAllSpecsRequired(t *testing.T) {
	// Format requires both 1080p AND WebDL.
	cf := customformats.CustomFormat{
		Name: "1080p WebDL",
		Specifications: []customformats.Specification{
			{
				Implementation: "ReleaseTitleSpecification",
				Value:          `(?i)\b1080p\b`,
			},
			{
				Implementation: "SourceSpecification",
				Value:          "webdl",
			},
		},
	}

	// Both specs satisfied: 1080p + webdl
	both := parser.ParsedEpisodeInfo{
		ReleaseTitle: "Show.S01E01.1080p.WEB-DL",
		Quality: parser.ParsedQuality{
			Resolution: parser.Resolution1080p,
			Source:     parser.SourceWebDL,
		},
	}
	if !customformats.Match(both, cf) {
		t.Error("expected 1080p WEB-DL to match both specs")
	}

	// Only resolution matches; source is BluRay.
	onlyRes := parser.ParsedEpisodeInfo{
		ReleaseTitle: "Show.S01E01.1080p.BluRay",
		Quality: parser.ParsedQuality{
			Resolution: parser.Resolution1080p,
			Source:     parser.SourceBluray,
		},
	}
	if customformats.Match(onlyRes, cf) {
		t.Error("expected 1080p BluRay NOT to match (source spec fails)")
	}

	// Only source matches; resolution is 720p.
	onlySrc := parser.ParsedEpisodeInfo{
		ReleaseTitle: "Show.S01E01.720p.WEB-DL",
		Quality: parser.ParsedQuality{
			Resolution: parser.Resolution720p,
			Source:     parser.SourceWebDL,
		},
	}
	if customformats.Match(onlySrc, cf) {
		t.Error("expected 720p WEB-DL NOT to match (title spec fails for 1080p)")
	}
}

func TestMatchSourceSpecification(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "BluRay",
		Specifications: []customformats.Specification{
			{Implementation: "SourceSpecification", Value: "bluray"},
		},
	}

	info := parser.ParsedEpisodeInfo{
		Quality: parser.ParsedQuality{Source: parser.SourceBluray},
	}
	if !customformats.Match(info, cf) {
		t.Error("expected bluray source to match SourceSpecification")
	}

	info.Quality.Source = parser.SourceWebDL
	if customformats.Match(info, cf) {
		t.Error("expected webdl source not to match bluray SourceSpecification")
	}
}

func TestMatchResolutionSpecification(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "2160p",
		Specifications: []customformats.Specification{
			{Implementation: "ResolutionSpecification", Value: "2160p"},
		},
	}

	info := parser.ParsedEpisodeInfo{
		Quality: parser.ParsedQuality{Resolution: parser.Resolution2160p},
	}
	if !customformats.Match(info, cf) {
		t.Error("expected 2160p resolution to match ResolutionSpecification")
	}

	info.Quality.Resolution = parser.Resolution1080p
	if customformats.Match(info, cf) {
		t.Error("expected 1080p not to match 2160p ResolutionSpecification")
	}
}

func TestMatchReleaseGroupSpecification(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "Group YIFY",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseGroupSpecification", Value: `(?i)^YIFY$`},
		},
	}

	info := parser.ParsedEpisodeInfo{ReleaseGroup: "YIFY"}
	if !customformats.Match(info, cf) {
		t.Error("expected YIFY release group to match")
	}

	info.ReleaseGroup = "SPARKS"
	if customformats.Match(info, cf) {
		t.Error("expected SPARKS not to match YIFY group spec")
	}
}

func TestMatchUnknownImplementationAccepts(t *testing.T) {
	// Unknown implementations should pass through (return true) so new spec
	// types don't break formats when they are added later.
	cf := customformats.CustomFormat{
		Name: "Future Spec",
		Specifications: []customformats.Specification{
			{Implementation: "SomeFutureSpecification", Value: "anything"},
		},
	}

	info := parser.ParsedEpisodeInfo{ReleaseTitle: "any title"}
	if !customformats.Match(info, cf) {
		t.Error("unknown implementation should be treated as matching (pass-through)")
	}
}

func TestMatchInvalidRegexDoesNotMatch(t *testing.T) {
	cf := customformats.CustomFormat{
		Name: "Bad Regex",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTitleSpecification", Value: `[invalid(`},
		},
	}

	info := parser.ParsedEpisodeInfo{ReleaseTitle: "any title"}
	if customformats.Match(info, cf) {
		t.Error("invalid regex should fail to match (not panic)")
	}
}

func TestMatchEmptyFormat(t *testing.T) {
	// A format with no specifications matches everything.
	cf := customformats.CustomFormat{Name: "Empty", Specifications: nil}
	info := parser.ParsedEpisodeInfo{}
	if !customformats.Match(info, cf) {
		t.Error("format with no specs should match everything")
	}
}

// --- Scorer tests ---

func TestScoreMultipleFormats(t *testing.T) {
	remux := customformats.CustomFormat{
		ID:   1,
		Name: "Remux",
		Specifications: []customformats.Specification{
			{Implementation: "SourceSpecification", Value: "remux"},
		},
	}
	hdr := customformats.CustomFormat{
		ID:   2,
		Name: "HDR",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTitleSpecification", Value: `(?i)\bHDR\b`},
		},
	}
	nonMatching := customformats.CustomFormat{
		ID:   3,
		Name: "Anime",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTitleSpecification", Value: `(?i)\bAnime\b`},
		},
	}

	profile := profiles.QualityProfile{
		FormatItems: []profiles.FormatScoreItem{
			{FormatID: 1, Score: 25},
			{FormatID: 2, Score: 15},
			{FormatID: 3, Score: 50},
		},
	}

	info := parser.ParsedEpisodeInfo{
		ReleaseTitle: "Show.S01E01.2160p.REMUX.HDR",
		Quality: parser.ParsedQuality{
			Source:     parser.SourceRemux,
			Resolution: parser.Resolution2160p,
		},
	}

	formats := []customformats.CustomFormat{remux, hdr, nonMatching}
	score := customformats.Score(info, formats, profile)

	// remux matches (+25) + hdr matches (+15) = 40; anime doesn't match (+0)
	if score != 40 {
		t.Errorf("score = %d, want 40", score)
	}
}

func TestScoreFormatNotInProfile(t *testing.T) {
	// A format that matches but has no weight entry contributes 0.
	cf := customformats.CustomFormat{
		ID:   99,
		Name: "Unweighted",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTitleSpecification", Value: `.*`},
		},
	}

	profile := profiles.QualityProfile{
		FormatItems: []profiles.FormatScoreItem{}, // no weights
	}
	info := parser.ParsedEpisodeInfo{ReleaseTitle: "anything"}
	score := customformats.Score(info, []customformats.CustomFormat{cf}, profile)
	if score != 0 {
		t.Errorf("score = %d, want 0", score)
	}
}

func TestScoreNegativeWeight(t *testing.T) {
	cf := customformats.CustomFormat{
		ID:   10,
		Name: "CAM",
		Specifications: []customformats.Specification{
			{Implementation: "ReleaseTitleSpecification", Value: `(?i)\bCAM\b`},
		},
	}

	profile := profiles.QualityProfile{
		FormatItems: []profiles.FormatScoreItem{
			{FormatID: 10, Score: -100},
		},
	}
	info := parser.ParsedEpisodeInfo{ReleaseTitle: "Show.S01E01.CAM"}
	score := customformats.Score(info, []customformats.CustomFormat{cf}, profile)
	if score != -100 {
		t.Errorf("score = %d, want -100", score)
	}
}

// --- Store CRUD tests ---

func newTestStore(t *testing.T) customformats.Store {
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
	return customformats.NewSQLiteStore(pool)
}

func TestStoreCRUD(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	in := customformats.CustomFormat{
		Name:                "Test Format",
		IncludeWhenRenaming: true,
		Specifications: []customformats.Specification{
			{
				Name:           "Title",
				Implementation: "ReleaseTitleSpecification",
				Value:          `(?i)\bREMUX\b`,
				Negate:         false,
				Required:       true,
			},
		},
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

	// GetByID
	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("got.Name = %q, want %q", got.Name, in.Name)
	}
	if got.IncludeWhenRenaming != in.IncludeWhenRenaming {
		t.Errorf("IncludeWhenRenaming = %v, want %v", got.IncludeWhenRenaming, in.IncludeWhenRenaming)
	}
	if len(got.Specifications) != 1 {
		t.Fatalf("len(Specifications) = %d, want 1", len(got.Specifications))
	}
	if got.Specifications[0].Value != in.Specifications[0].Value {
		t.Errorf("spec value = %q, want %q", got.Specifications[0].Value, in.Specifications[0].Value)
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
	got.Name = "Updated Format"
	got.IncludeWhenRenaming = false
	got.Specifications = append(got.Specifications, customformats.Specification{
		Implementation: "SourceSpecification",
		Value:          "bluray",
	})
	if err := store.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, err := store.GetByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.Name != "Updated Format" {
		t.Errorf("Name after update = %q, want Updated Format", after.Name)
	}
	if after.IncludeWhenRenaming {
		t.Error("IncludeWhenRenaming after update = true, want false")
	}
	if len(after.Specifications) != 2 {
		t.Errorf("len(Specifications) after update = %d, want 2", len(after.Specifications))
	}

	// Delete
	if err := store.Delete(ctx, got.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.GetByID(ctx, got.ID)
	if !errors.Is(err, customformats.ErrNotFound) {
		t.Errorf("GetByID after delete = %v, want ErrNotFound", err)
	}
}

func TestStoreGetByIDNotFound(t *testing.T) {
	store := newTestStore(t)
	_, err := store.GetByID(context.Background(), 9999)
	if !errors.Is(err, customformats.ErrNotFound) {
		t.Errorf("GetByID(missing) = %v, want ErrNotFound", err)
	}
}

func TestStoreSpecificationsJSONRoundtrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	specs := []customformats.Specification{
		{Name: "Title", Implementation: "ReleaseTitleSpecification", Value: `REMUX`, Negate: false, Required: true},
		{Name: "Source", Implementation: "SourceSpecification", Value: "remux", Negate: false, Required: true},
		{Name: "Not HDTV", Implementation: "SourceSpecification", Value: "television", Negate: true, Required: false},
	}

	created, err := store.Create(ctx, customformats.CustomFormat{
		Name:           "Complex Format",
		Specifications: specs,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if len(got.Specifications) != len(specs) {
		t.Fatalf("Specifications len = %d, want %d", len(got.Specifications), len(specs))
	}
	for i, want := range specs {
		g := got.Specifications[i]
		if g.Name != want.Name || g.Implementation != want.Implementation ||
			g.Value != want.Value || g.Negate != want.Negate || g.Required != want.Required {
			t.Errorf("Specifications[%d] = %+v, want %+v", i, g, want)
		}
	}
}
