package customformats

import (
	"github.com/ajthom90/sonarr2/internal/parser"
	"github.com/ajthom90/sonarr2/internal/profiles"
)

// Score computes the total custom format score for a release against a quality
// profile's format weights. For each custom format that matches the release,
// the corresponding weight from profile.FormatItems is added to the total. If
// a matching format has no entry in profile.FormatItems, it contributes 0.
func Score(info parser.ParsedEpisodeInfo, formats []CustomFormat, profile profiles.QualityProfile) int {
	// Build a map of formatID → score for fast lookup.
	weights := make(map[int]int, len(profile.FormatItems))
	for _, fi := range profile.FormatItems {
		weights[fi.FormatID] = fi.Score
	}

	total := 0
	for _, cf := range formats {
		if Match(info, cf) {
			total += weights[cf.ID]
		}
	}
	return total
}
