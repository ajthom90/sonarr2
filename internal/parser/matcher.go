package parser

import (
	"context"
	"strings"
)

// SeriesLookup resolves a parsed series title to a series ID. Implemented
// by the library package in the composition root — the parser package
// itself has no DB dependency.
type SeriesLookup interface {
	FindByTitle(ctx context.Context, title string) (seriesID int64, found bool, err error)
}

// MatchResult pairs a ParsedEpisodeInfo with the matched series ID.
type MatchResult struct {
	SeriesID int64
	Info     ParsedEpisodeInfo
	Matched  bool
}

// MatchSeries attempts to match a parsed title against the series database.
// It normalizes the title and queries the lookup. Returns Matched=false if
// no series was found (not an error — unmatched releases are expected).
func MatchSeries(ctx context.Context, lookup SeriesLookup, info ParsedEpisodeInfo) (MatchResult, error) {
	normalized := normalizeForLookup(info.SeriesTitle)
	id, found, err := lookup.FindByTitle(ctx, normalized)
	if err != nil {
		return MatchResult{}, err
	}
	return MatchResult{
		SeriesID: id,
		Info:     info,
		Matched:  found,
	}, nil
}

// normalizeForLookup lowercases and trims a title for lookup.
func normalizeForLookup(title string) string {
	return strings.ToLower(strings.TrimSpace(title))
}
