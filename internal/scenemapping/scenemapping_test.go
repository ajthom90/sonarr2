package scenemapping_test

import (
	"testing"

	"github.com/ajthom90/sonarr2/internal/scenemapping"
)

func TestLookupSceneSeason(t *testing.T) {
	two := 2
	three := 3
	mappings := []scenemapping.Mapping{
		{TvdbID: 100, SeasonNumber: &two, SceneSeasonNumber: &three},
		{TvdbID: 200, SeasonNumber: nil, SceneSeasonNumber: &three}, // any season → 3
	}

	if got := scenemapping.LookupSceneSeason(mappings, 100, 2); got != 3 {
		t.Errorf("scene season = %d, want 3", got)
	}
	if got := scenemapping.LookupSceneSeason(mappings, 100, 1); got != 1 {
		t.Errorf("no mapping for S1, should return input %d, got %d", 1, got)
	}
	if got := scenemapping.LookupSceneSeason(mappings, 200, 5); got != 3 {
		t.Errorf("all-seasons mapping should apply; got %d", got)
	}
	if got := scenemapping.LookupSceneSeason(mappings, 999, 1); got != 1 {
		t.Errorf("unknown series should return input; got %d", got)
	}
}

func TestLookupTvdbIDByTitle(t *testing.T) {
	mappings := []scenemapping.Mapping{
		{TvdbID: 100, ParseTerm: "Attack on Titan", Title: "Shingeki no Kyojin"},
		{TvdbID: 200, ParseTerm: "Naruto Shippuden"},
	}
	id, ok := scenemapping.LookupTvdbIDByTitle(mappings, "attack-on-titan")
	if !ok || id != 100 {
		t.Errorf("want (100,true), got (%d,%v)", id, ok)
	}
	_, ok = scenemapping.LookupTvdbIDByTitle(mappings, "unknown series")
	if ok {
		t.Error("expected no match for unknown title")
	}
}
