package organizer

// Token patterns that get replaced in the naming format string.
// Example format: "{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality Full}]"
const (
	TokenSeriesTitle  = "{Series Title}"
	TokenSeason       = "{season:00}"
	TokenEpisode      = "{episode:00}"
	TokenEpisodeTitle = "{Episode Title}"
	TokenQualityFull  = "{Quality Full}"
	TokenReleaseGroup = "{Release Group}"
	TokenAirDate      = "{Air-Date}"
)
