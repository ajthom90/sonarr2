package parser

import (
	"context"
	"testing"
)

type stubLookup struct {
	series map[string]int64
}

func (s *stubLookup) FindByTitle(_ context.Context, title string) (int64, bool, error) {
	id, ok := s.series[title]
	return id, ok, nil
}

func TestMatchSeriesFound(t *testing.T) {
	lookup := &stubLookup{
		series: map[string]int64{
			"the simpsons": 42,
		},
	}
	info := ParseTitle("The.Simpsons.S35E10.1080p.WEB-DL.x264-GROUP")
	result, err := MatchSeries(context.Background(), lookup, info)
	if err != nil {
		t.Fatalf("MatchSeries returned error: %v", err)
	}
	if !result.Matched {
		t.Errorf("Matched = false, want true")
	}
	if result.SeriesID != 42 {
		t.Errorf("SeriesID = %d, want 42", result.SeriesID)
	}
}

func TestMatchSeriesNotFound(t *testing.T) {
	lookup := &stubLookup{
		series: map[string]int64{},
	}
	info := ParseTitle("The.Simpsons.S35E10.1080p.WEB-DL.x264-GROUP")
	result, err := MatchSeries(context.Background(), lookup, info)
	if err != nil {
		t.Fatalf("MatchSeries returned error: %v", err)
	}
	if result.Matched {
		t.Errorf("Matched = true, want false")
	}
	if result.SeriesID != 0 {
		t.Errorf("SeriesID = %d, want 0", result.SeriesID)
	}
}

func TestNormalizeForLookup(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"The Simpsons", "the simpsons"},
		{"  SHOW  ", "show"},
		{"Breaking Bad", "breaking bad"},
		{"", ""},
	}
	for _, tc := range cases {
		got := normalizeForLookup(tc.input)
		if got != tc.want {
			t.Errorf("normalizeForLookup(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
