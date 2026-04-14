// Package scenemapping caches anime scene-number → absolute-number mappings
// fetched from a remote source (Sonarr historically pulled from the
// skyhook mapping list). Applied during release parsing so anime releases
// that use absolute episode numbers can be resolved to Sonarr's
// season/episode scheme.
//
// Ported architecturally from Sonarr
// (src/NzbDrone.Core/DataAugmentation/Scene/).
package scenemapping

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned when no mapping matches.
var ErrNotFound = errors.New("scenemapping: not found")

// Mapping is one row in scene_mappings. SeasonNumber / SceneSeasonNumber
// are optional — a mapping with a nil SeasonNumber applies to every season.
type Mapping struct {
	ID                int
	TvdbID            int64
	SeasonNumber      *int
	SceneSeasonNumber *int
	SceneOrigin       string
	Comment           string
	FilterRegex       string
	ParseTerm         string
	SearchTerm        string
	Title             string
	Type              string
	UpdatedAt         time.Time
}

// Store persists scene mappings.
type Store interface {
	ReplaceAll(ctx context.Context, mappings []Mapping) error
	ListByTvdbID(ctx context.Context, tvdbID int64) ([]Mapping, error)
	ListAll(ctx context.Context) ([]Mapping, error)
}

// LookupSceneSeason returns the scene season number for a given
// tvdb series + native season, or the native season if no mapping applies.
func LookupSceneSeason(mappings []Mapping, tvdbID int64, season int) int {
	for _, m := range mappings {
		if m.TvdbID != tvdbID {
			continue
		}
		if m.SeasonNumber == nil || *m.SeasonNumber == season {
			if m.SceneSeasonNumber != nil {
				return *m.SceneSeasonNumber
			}
		}
	}
	return season
}

// LookupTvdbIDByTitle finds a tvdb ID whose ParseTerm matches the given
// parsed release title. Used during release parsing for anime releases
// that only carry the scene title.
func LookupTvdbIDByTitle(mappings []Mapping, parsedTitle string) (int64, bool) {
	for _, m := range mappings {
		if m.ParseTerm == "" {
			continue
		}
		if normalize(m.ParseTerm) == normalize(parsedTitle) {
			return m.TvdbID, true
		}
	}
	return 0, false
}

// normalize lowercases and collapses runs of non-alphanumeric characters.
// Matches Sonarr's scene-title normalization.
func normalize(s string) string {
	out := make([]byte, 0, len(s))
	lastSep := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			out = append(out, c)
			lastSep = false
		} else if !lastSep {
			out = append(out, ' ')
			lastSep = true
		}
	}
	// Trim trailing separator.
	if len(out) > 0 && out[len(out)-1] == ' ' {
		out = out[:len(out)-1]
	}
	return string(out)
}
