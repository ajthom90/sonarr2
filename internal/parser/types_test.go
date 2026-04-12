package parser

import (
	"testing"
	"time"
)

func TestQualitySourceConstants(t *testing.T) {
	cases := []struct {
		name string
		got  QualitySource
		want string
	}{
		{"SourceUnknown", SourceUnknown, ""},
		{"SourceTelevision", SourceTelevision, "television"},
		{"SourceWebDL", SourceWebDL, "webdl"},
		{"SourceWebRip", SourceWebRip, "webrip"},
		{"SourceBluray", SourceBluray, "bluray"},
		{"SourceDVD", SourceDVD, "dvd"},
		{"SourceRemux", SourceRemux, "remux"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.got) != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestResolutionConstants(t *testing.T) {
	cases := []struct {
		name string
		got  Resolution
		want string
	}{
		{"ResolutionUnknown", ResolutionUnknown, ""},
		{"Resolution480p", Resolution480p, "480p"},
		{"Resolution576p", Resolution576p, "576p"},
		{"Resolution720p", Resolution720p, "720p"},
		{"Resolution1080p", Resolution1080p, "1080p"},
		{"Resolution2160p", Resolution2160p, "2160p"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.got) != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestModifierConstants(t *testing.T) {
	cases := []struct {
		name string
		got  Modifier
		want string
	}{
		{"ModifierNone", ModifierNone, ""},
		{"ModifierProper", ModifierProper, "proper"},
		{"ModifierRepack", ModifierRepack, "repack"},
		{"ModifierReal", ModifierReal, "real"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.got) != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestParsedQualityUsable(t *testing.T) {
	q := ParsedQuality{
		Source:     SourceWebDL,
		Resolution: Resolution1080p,
		Modifier:   ModifierProper,
		Revision:   2,
	}
	if q.Source != SourceWebDL {
		t.Errorf("Source = %q, want %q", q.Source, SourceWebDL)
	}
	if q.Resolution != Resolution1080p {
		t.Errorf("Resolution = %q, want %q", q.Resolution, Resolution1080p)
	}
	if q.Modifier != ModifierProper {
		t.Errorf("Modifier = %q, want %q", q.Modifier, ModifierProper)
	}
	if q.Revision != 2 {
		t.Errorf("Revision = %d, want 2", q.Revision)
	}
}

func TestParsedEpisodeInfoUsable(t *testing.T) {
	now := time.Now()
	info := ParsedEpisodeInfo{
		SeriesTitle:            "The Simpsons",
		SeriesType:             SeriesTypeStandard,
		SeasonNumber:           35,
		EpisodeNumbers:         []int{10},
		AbsoluteEpisodeNumbers: []int{},
		AirDate:                &now,
		Year:                   2024,
		ReleaseGroup:           "GROUP",
		Quality:                ParsedQuality{Source: SourceWebDL, Resolution: Resolution1080p, Revision: 1},
		ReleaseTitle:           "The.Simpsons.S35E10.1080p.WEB-DL.x264-GROUP",
	}
	if info.SeriesTitle != "The Simpsons" {
		t.Errorf("SeriesTitle = %q, want %q", info.SeriesTitle, "The Simpsons")
	}
	if info.SeriesType != SeriesTypeStandard {
		t.Errorf("SeriesType = %q, want %q", info.SeriesType, SeriesTypeStandard)
	}
	if info.SeasonNumber != 35 {
		t.Errorf("SeasonNumber = %d, want 35", info.SeasonNumber)
	}
	if len(info.EpisodeNumbers) != 1 || info.EpisodeNumbers[0] != 10 {
		t.Errorf("EpisodeNumbers = %v, want [10]", info.EpisodeNumbers)
	}
}

func TestSeriesTypeConstants(t *testing.T) {
	cases := []struct {
		name string
		got  SeriesType
		want string
	}{
		{"SeriesTypeStandard", SeriesTypeStandard, "standard"},
		{"SeriesTypeDaily", SeriesTypeDaily, "daily"},
		{"SeriesTypeAnime", SeriesTypeAnime, "anime"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if string(tc.got) != tc.want {
				t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
			}
		})
	}
}
