package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Standard: Show.Name.S01E01, Show.Name.S01E01E02, Show.Name.S01E01-E03
	reStandard = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` + // series title (lazy)
			`[\.\s_-]+` + // separator
			`S(?P<season>\d{1,2})` + // S01
			`E(?P<episode>\d{2,3})` + // E01
			`(?:[-E]+(?P<episode2>\d{2,3}))*`, // optional E02, E03, -E03
	)

	// Daily: Show.Name.2024.03.15
	reDaily = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` +
			`[\.\s_-]+` +
			`(?P<year>\d{4})[\.\s_-]` +
			`(?P<month>\d{2})[\.\s_-]` +
			`(?P<day>\d{2})`,
	)

	// Anime absolute: [Group] Show Name - 01, [Group] Show - 123v2
	reAnime = regexp.MustCompile(
		`(?i)^\[(?P<group>[^\]]+)\]\s*` + // [Group]
			`(?P<title>.+?)` + // title
			`\s*-\s*` + // separator
			`(?P<episode>\d{2,4})` + // absolute episode number
			`(?:v\d+)?`, // optional version
	)

	// Fallback: just grab everything before common quality/source tokens
	reFallbackTitle = regexp.MustCompile(
		`(?i)^(?P<title>.+?)` +
			`(?:[\.\s_-]+(?:S\d|E\d|\d{4}[\.\s_-]\d{2}|\d{3,4}p|HDTV|WEB|BluRay|DVD|x264|x265|H\.?264|H\.?265|AAC|AC3|DTS|PROPER|REPACK))`,
	)

	// Release group: -GROUP at the end
	reReleaseGroup = regexp.MustCompile(`-([A-Za-z0-9]+)(?:\[.*\])?$`)
)

// ParseTitle parses a release title into structured episode information.
// It tries standard (SxxExx), daily (YYYY.MM.DD), and anime (absolute)
// patterns in order, falling back to title-only extraction if none match.
func ParseTitle(title string) ParsedEpisodeInfo {
	info := ParsedEpisodeInfo{
		ReleaseTitle: title,
		Quality:      ParseQuality(title),
	}

	// Try standard first (most common).
	if parseStandard(title, &info) {
		info.SeriesType = SeriesTypeStandard
		info.ReleaseGroup = parseReleaseGroup(title)
		return info
	}

	// Try daily.
	if parseDaily(title, &info) {
		info.SeriesType = SeriesTypeDaily
		info.ReleaseGroup = parseReleaseGroup(title)
		return info
	}

	// Try anime.
	if parseAnime(title, &info) {
		info.SeriesType = SeriesTypeAnime
		return info
	}

	// Fallback: extract title only.
	if m := reFallbackTitle.FindStringSubmatch(title); len(m) > 1 {
		info.SeriesTitle = cleanTitle(m[1])
	} else {
		info.SeriesTitle = cleanTitle(title)
	}
	info.ReleaseGroup = parseReleaseGroup(title)
	return info
}

func parseStandard(title string, info *ParsedEpisodeInfo) bool {
	m := reStandard.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	names := reStandard.SubexpNames()
	fields := make(map[string]string)
	for i, name := range names {
		if name != "" && i < len(m) {
			fields[name] = m[i]
		}
	}

	info.SeriesTitle = cleanTitle(fields["title"])
	info.SeasonNumber, _ = strconv.Atoi(fields["season"])
	if ep, err := strconv.Atoi(fields["episode"]); err == nil {
		info.EpisodeNumbers = append(info.EpisodeNumbers, ep)
	}
	if ep2, err := strconv.Atoi(fields["episode2"]); err == nil && ep2 > 0 {
		// Range: fill in episodes between first and last.
		first := info.EpisodeNumbers[0]
		info.EpisodeNumbers = nil
		for e := first; e <= ep2; e++ {
			info.EpisodeNumbers = append(info.EpisodeNumbers, e)
		}
	}
	return len(info.EpisodeNumbers) > 0
}

func parseDaily(title string, info *ParsedEpisodeInfo) bool {
	m := reDaily.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	names := reDaily.SubexpNames()
	fields := make(map[string]string)
	for i, name := range names {
		if name != "" && i < len(m) {
			fields[name] = m[i]
		}
	}

	info.SeriesTitle = cleanTitle(fields["title"])
	year, _ := strconv.Atoi(fields["year"])
	month, _ := strconv.Atoi(fields["month"])
	day, _ := strconv.Atoi(fields["day"])
	info.Year = year

	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	info.AirDate = &t
	info.SeasonNumber = year
	info.EpisodeNumbers = []int{month*100 + day} // MMDD as episode number for daily
	return true
}

func parseAnime(title string, info *ParsedEpisodeInfo) bool {
	m := reAnime.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	names := reAnime.SubexpNames()
	fields := make(map[string]string)
	for i, name := range names {
		if name != "" && i < len(m) {
			fields[name] = m[i]
		}
	}

	info.SeriesTitle = cleanTitle(fields["title"])
	info.ReleaseGroup = strings.TrimSpace(fields["group"])
	if ep, err := strconv.Atoi(fields["episode"]); err == nil {
		info.AbsoluteEpisodeNumbers = append(info.AbsoluteEpisodeNumbers, ep)
	}
	return len(info.AbsoluteEpisodeNumbers) > 0
}

func parseReleaseGroup(title string) string {
	m := reReleaseGroup.FindStringSubmatch(title)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// cleanTitle normalizes a parsed series title: replaces dots/underscores
// with spaces, trims whitespace.
func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, ".", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	// Collapse multiple spaces.
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return s
}
