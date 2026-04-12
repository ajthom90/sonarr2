package parser

import "time"

// QualitySource identifies where the media came from.
type QualitySource string

const (
	SourceUnknown    QualitySource = ""
	SourceTelevision QualitySource = "television" // HDTV, PDTV, SDTV
	SourceWebDL      QualitySource = "webdl"      // WEB-DL, WEBDL, WEB.DL
	SourceWebRip     QualitySource = "webrip"     // WEBRip
	SourceBluray     QualitySource = "bluray"     // Bluray, BDRip, BRRip
	SourceDVD        QualitySource = "dvd"        // DVDRip, DVD-R, DVDSCR
	SourceRemux      QualitySource = "remux"      // Remux, BDREMUX
)

// Resolution identifies the video resolution.
type Resolution string

const (
	ResolutionUnknown Resolution = ""
	Resolution480p    Resolution = "480p"
	Resolution576p    Resolution = "576p"
	Resolution720p    Resolution = "720p"
	Resolution1080p   Resolution = "1080p"
	Resolution2160p   Resolution = "2160p"
)

// Modifier identifies quality modifiers.
type Modifier string

const (
	ModifierNone   Modifier = ""
	ModifierProper Modifier = "proper"
	ModifierRepack Modifier = "repack"
	ModifierReal   Modifier = "real"
)

// ParsedQuality is the result of quality parsing.
type ParsedQuality struct {
	Source     QualitySource
	Resolution Resolution
	Modifier   Modifier
	Revision   int // 1 = original, 2+ = anime v2/v3
}

// SeriesType distinguishes numbering conventions.
type SeriesType string

const (
	SeriesTypeStandard SeriesType = "standard"
	SeriesTypeDaily    SeriesType = "daily"
	SeriesTypeAnime    SeriesType = "anime"
)

// ParsedEpisodeInfo is the result of title parsing.
type ParsedEpisodeInfo struct {
	SeriesTitle            string
	SeriesType             SeriesType
	SeasonNumber           int
	EpisodeNumbers         []int
	AbsoluteEpisodeNumbers []int
	AirDate                *time.Time // non-nil for daily series
	Year                   int
	ReleaseGroup           string
	Quality                ParsedQuality
	// ReleaseTitle is the original unparsed title, kept for logging.
	ReleaseTitle string
}
