package organizer

import (
	"fmt"
	"strings"
)

// EpisodeInfo holds the metadata needed to build a destination filename.
type EpisodeInfo struct {
	SeriesTitle   string
	SeasonNumber  int
	EpisodeNumber int
	EpisodeTitle  string
	QualityName   string
	ReleaseGroup  string
	AirDate       string // YYYY-MM-DD
}

// DefaultEpisodeFormat is the standard naming format.
const DefaultEpisodeFormat = "{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality Full}]"

// BuildFilename applies the naming format to episode info, producing
// a filename (without extension).
func BuildFilename(format string, info EpisodeInfo) string {
	// Replace each token with the actual value.
	result := format
	result = strings.ReplaceAll(result, TokenSeriesTitle, info.SeriesTitle)
	result = strings.ReplaceAll(result, TokenSeason, fmt.Sprintf("%02d", info.SeasonNumber))
	result = strings.ReplaceAll(result, TokenEpisode, fmt.Sprintf("%02d", info.EpisodeNumber))
	result = strings.ReplaceAll(result, TokenEpisodeTitle, info.EpisodeTitle)
	result = strings.ReplaceAll(result, TokenQualityFull, info.QualityName)
	result = strings.ReplaceAll(result, TokenReleaseGroup, info.ReleaseGroup)
	result = strings.ReplaceAll(result, TokenAirDate, info.AirDate)
	// Clean illegal filename characters.
	return cleanFilename(result)
}

// BuildSeasonFolder returns the season subfolder name.
func BuildSeasonFolder(seasonNumber int) string {
	return fmt.Sprintf("Season %02d", seasonNumber)
}

func cleanFilename(name string) string {
	// Remove characters illegal in most filesystems.
	for _, c := range []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"} {
		name = strings.ReplaceAll(name, c, "")
	}
	return strings.TrimSpace(name)
}
