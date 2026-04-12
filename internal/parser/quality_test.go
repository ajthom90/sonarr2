package parser

import "testing"

func TestParseQuality(t *testing.T) {
	cases := []struct {
		title string
		want  ParsedQuality
	}{
		{"Show.S01E01.1080p.WEB-DL.x264-GROUP", ParsedQuality{Source: SourceWebDL, Resolution: Resolution1080p, Revision: 1}},
		{"Show.S01E01.720p.HDTV.x264-GROUP", ParsedQuality{Source: SourceTelevision, Resolution: Resolution720p, Revision: 1}},
		{"Show.S01E01.2160p.BluRay.REMUX-GROUP", ParsedQuality{Source: SourceRemux, Resolution: Resolution2160p, Revision: 1}},
		{"Show.S01E01.480p.DVDRip-GROUP", ParsedQuality{Source: SourceDVD, Resolution: Resolution480p, Revision: 1}},
		{"Show.S01E01.WEBRip.x264-GROUP", ParsedQuality{Source: SourceWebRip, Revision: 1}},
		{"Show.S01E01.1080p.BluRay.x264-GROUP", ParsedQuality{Source: SourceBluray, Resolution: Resolution1080p, Revision: 1}},
		{"Show.S01E01.1080p.WEB-DL.PROPER-GROUP", ParsedQuality{Source: SourceWebDL, Resolution: Resolution1080p, Modifier: ModifierProper, Revision: 1}},
		{"Show.S01E01.REPACK.720p.HDTV-GROUP", ParsedQuality{Source: SourceTelevision, Resolution: Resolution720p, Modifier: ModifierRepack, Revision: 1}},
		{"[SubGroup] Show - 01v2 [1080p]", ParsedQuality{Resolution: Resolution1080p, Revision: 2}},
		{"nothing parseable here", ParsedQuality{Revision: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			got := ParseQuality(tc.title)
			if got != tc.want {
				t.Errorf("ParseQuality(%q)\n  got  %+v\n  want %+v", tc.title, got, tc.want)
			}
		})
	}
}
