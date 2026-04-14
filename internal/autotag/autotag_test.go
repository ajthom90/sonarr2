package autotag_test

import (
	"testing"

	"github.com/ajthom90/sonarr2/internal/autotag"
)

func TestMatchesRequiredAnd(t *testing.T) {
	rule := autotag.Rule{
		Specifications: []autotag.Specification{
			{Implementation: "GenreSpecification", Value: "Comedy", Required: true},
			{Implementation: "SeriesStatusSpecification", Value: "continuing", Required: true},
		},
		Tags: []int{1},
	}

	good := autotag.SeriesAttr{Genres: []string{"Drama", "Comedy"}, Status: "continuing"}
	if !autotag.Matches(rule, good) {
		t.Error("expected match (both required met)")
	}

	bad := autotag.SeriesAttr{Genres: []string{"Drama"}, Status: "continuing"}
	if autotag.Matches(rule, bad) {
		t.Error("expected no match when one required missing")
	}
}

func TestMatchesNegate(t *testing.T) {
	rule := autotag.Rule{
		Specifications: []autotag.Specification{
			{Implementation: "GenreSpecification", Value: "Reality-TV", Required: true, Negate: true},
		},
	}
	if !autotag.Matches(rule, autotag.SeriesAttr{Genres: []string{"Drama"}}) {
		t.Error("expected match (reality-tv absent, negated required)")
	}
	if autotag.Matches(rule, autotag.SeriesAttr{Genres: []string{"Reality-TV"}}) {
		t.Error("expected no match (reality-tv present, negated required fails)")
	}
}

func TestMatchesOptionalOr(t *testing.T) {
	rule := autotag.Rule{
		Specifications: []autotag.Specification{
			{Implementation: "SeriesTypeSpecification", Value: "anime"},
			{Implementation: "GenreSpecification", Value: "Animation"},
		},
	}
	// Either matches.
	if !autotag.Matches(rule, autotag.SeriesAttr{SeriesType: "anime"}) {
		t.Error("expected match (type=anime)")
	}
	if !autotag.Matches(rule, autotag.SeriesAttr{Genres: []string{"Animation"}}) {
		t.Error("expected match (genre=Animation)")
	}
	if autotag.Matches(rule, autotag.SeriesAttr{Genres: []string{"Drama"}}) {
		t.Error("expected no match")
	}
}

func TestYearRange(t *testing.T) {
	rule := autotag.Rule{
		Specifications: []autotag.Specification{
			{Implementation: "YearSpecification", Value: "2020-2024", Required: true},
		},
	}
	if !autotag.Matches(rule, autotag.SeriesAttr{Year: 2022}) {
		t.Error("expected match for year in range")
	}
	if autotag.Matches(rule, autotag.SeriesAttr{Year: 2019}) {
		t.Error("expected no match below range")
	}
}

func TestApplyAndRemoveTags(t *testing.T) {
	existing := []int{1, 2}
	rule := autotag.Rule{Tags: []int{2, 3}}
	got := autotag.ApplyTags(existing, rule)
	if len(got) != 3 {
		t.Errorf("ApplyTags = %v, want [1 2 3]", got)
	}
	removed := autotag.RemoveTags([]int{1, 2, 3, 4}, rule)
	if len(removed) != 2 || removed[0] != 1 || removed[1] != 4 {
		t.Errorf("RemoveTags = %v, want [1 4]", removed)
	}
}
