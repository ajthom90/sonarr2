package parser

import (
	"regexp"
	"strconv"
)

var (
	// Resolution patterns
	reResolution2160p = regexp.MustCompile(`(?i)\b(2160p|4k|uhd)\b`)
	reResolution1080p = regexp.MustCompile(`(?i)\b1080[pi]\b`)
	reResolution720p  = regexp.MustCompile(`(?i)\b720p\b`)
	reResolution480p  = regexp.MustCompile(`(?i)\b480[pi]\b`)
	reResolution576p  = regexp.MustCompile(`(?i)\b576[pi]\b`)

	// Source patterns
	reWebDL  = regexp.MustCompile(`(?i)\b(WEB[-. ]?DL|WEBDL)\b`)
	reWebRip = regexp.MustCompile(`(?i)\b(WEB[-. ]?Rip|WEBRip)\b`)
	reBluray = regexp.MustCompile(`(?i)\b(Blu[-. ]?Ray|BD[-. ]?Rip|BR[-. ]?Rip|BDREMUX)\b`)
	reRemux  = regexp.MustCompile(`(?i)\b(REMUX|BDREMUX)\b`)
	reHDTV   = regexp.MustCompile(`(?i)\b(HDTV|PDTV|DSR|DTHD)\b`)
	reSDTV   = regexp.MustCompile(`(?i)\b(SDTV)\b`)
	reDVD    = regexp.MustCompile(`(?i)\b(DVD[-. ]?Rip|DVD[-. ]?R|DVDSCR)\b`)

	// Modifier patterns
	reProper = regexp.MustCompile(`(?i)\bPROPER\b`)
	reRepack = regexp.MustCompile(`(?i)\bREPACK\b`)
	reReal   = regexp.MustCompile(`(?i)\bREAL\b`)

	// Anime version pattern (v2, v3, etc.)
	// No leading \b because the version suffix is often attached to an episode number (e.g. 01v2).
	reAnimeVersion = regexp.MustCompile(`(?i)v(\d+)\b`)
)

// ParseQuality extracts quality information from a release title.
func ParseQuality(title string) ParsedQuality {
	q := ParsedQuality{Revision: 1}

	// Resolution
	switch {
	case reResolution2160p.MatchString(title):
		q.Resolution = Resolution2160p
	case reResolution1080p.MatchString(title):
		q.Resolution = Resolution1080p
	case reResolution720p.MatchString(title):
		q.Resolution = Resolution720p
	case reResolution576p.MatchString(title):
		q.Resolution = Resolution576p
	case reResolution480p.MatchString(title):
		q.Resolution = Resolution480p
	}

	// Source (order matters: check remux before bluray since BDREMUX matches both)
	switch {
	case reRemux.MatchString(title):
		q.Source = SourceRemux
	case reWebDL.MatchString(title):
		q.Source = SourceWebDL
	case reWebRip.MatchString(title):
		q.Source = SourceWebRip
	case reBluray.MatchString(title):
		q.Source = SourceBluray
	case reDVD.MatchString(title):
		q.Source = SourceDVD
	case reHDTV.MatchString(title):
		q.Source = SourceTelevision
	case reSDTV.MatchString(title):
		q.Source = SourceTelevision
	}

	// Modifier
	switch {
	case reProper.MatchString(title):
		q.Modifier = ModifierProper
	case reRepack.MatchString(title):
		q.Modifier = ModifierRepack
	case reReal.MatchString(title):
		q.Modifier = ModifierReal
	}

	// Anime version
	if m := reAnimeVersion.FindStringSubmatch(title); len(m) > 1 {
		if v, err := strconv.Atoi(m[1]); err == nil && v > 1 {
			q.Revision = v
		}
	}

	return q
}
