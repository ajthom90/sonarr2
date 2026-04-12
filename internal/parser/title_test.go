package parser

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseTitle(t *testing.T) {
	cases := []struct {
		name  string
		title string
		want  ParsedEpisodeInfo
	}{
		{
			"standard single episode",
			"The.Simpsons.S35E10.1080p.WEB-DL.x264-GROUP",
			ParsedEpisodeInfo{
				SeriesTitle:    "The Simpsons",
				SeriesType:     SeriesTypeStandard,
				SeasonNumber:   35,
				EpisodeNumbers: []int{10},
				ReleaseGroup:   "GROUP",
			},
		},
		{
			"standard multi-episode",
			"Show.Name.S02E03E04.720p.HDTV-LOL",
			ParsedEpisodeInfo{
				SeriesTitle:    "Show Name",
				SeriesType:     SeriesTypeStandard,
				SeasonNumber:   2,
				EpisodeNumbers: []int{3, 4},
				ReleaseGroup:   "LOL",
			},
		},
		{
			"daily show",
			"Jeopardy.2024.03.15.720p.HDTV-NTb",
			ParsedEpisodeInfo{
				SeriesTitle:    "Jeopardy",
				SeriesType:     SeriesTypeDaily,
				SeasonNumber:   2024,
				EpisodeNumbers: []int{315},
				Year:           2024,
				ReleaseGroup:   "NTb",
			},
		},
		{
			"anime absolute",
			"[SubGroup] One Piece - 1100 [1080p]",
			ParsedEpisodeInfo{
				SeriesTitle:            "One Piece",
				SeriesType:             SeriesTypeAnime,
				AbsoluteEpisodeNumbers: []int{1100},
				ReleaseGroup:           "SubGroup",
			},
		},
		{
			"anime with version",
			"[Erai-raws] Demon Slayer - 26v2 [720p]",
			ParsedEpisodeInfo{
				SeriesTitle:            "Demon Slayer",
				SeriesType:             SeriesTypeAnime,
				AbsoluteEpisodeNumbers: []int{26},
				ReleaseGroup:           "Erai-raws",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseTitle(tc.title)
			// Compare key fields (ignore Quality and ReleaseTitle — tested separately).
			if got.SeriesTitle != tc.want.SeriesTitle {
				t.Errorf("SeriesTitle = %q, want %q", got.SeriesTitle, tc.want.SeriesTitle)
			}
			if got.SeriesType != tc.want.SeriesType {
				t.Errorf("SeriesType = %q, want %q", got.SeriesType, tc.want.SeriesType)
			}
			if got.SeasonNumber != tc.want.SeasonNumber {
				t.Errorf("SeasonNumber = %d, want %d", got.SeasonNumber, tc.want.SeasonNumber)
			}
			if !slicesEqual(got.EpisodeNumbers, tc.want.EpisodeNumbers) {
				t.Errorf("EpisodeNumbers = %v, want %v", got.EpisodeNumbers, tc.want.EpisodeNumbers)
			}
			if !slicesEqual(got.AbsoluteEpisodeNumbers, tc.want.AbsoluteEpisodeNumbers) {
				t.Errorf("AbsoluteEpisodeNumbers = %v, want %v", got.AbsoluteEpisodeNumbers, tc.want.AbsoluteEpisodeNumbers)
			}
			if got.ReleaseGroup != tc.want.ReleaseGroup {
				t.Errorf("ReleaseGroup = %q, want %q", got.ReleaseGroup, tc.want.ReleaseGroup)
			}
			if got.Year != tc.want.Year {
				t.Errorf("Year = %d, want %d", got.Year, tc.want.Year)
			}
		})
	}
}

func TestCleanTitle(t *testing.T) {
	cases := map[string]string{
		"The.Simpsons":     "The Simpsons",
		"Show_Name":        "Show Name",
		"  Extra  Spaces ": "Extra Spaces",
		"Already Clean":    "Already Clean",
	}
	for in, want := range cases {
		if got := cleanTitle(in); got != want {
			t.Errorf("cleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
}

func slicesEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestParseTitleGoldenFile(t *testing.T) {
	data, err := os.ReadFile("testdata/titles.json")
	if err != nil {
		t.Fatalf("read golden file: %v", err)
	}
	var cases []struct {
		Title                  string `json:"title"`
		SeriesTitle            string `json:"seriesTitle"`
		SeriesType             string `json:"seriesType"`
		SeasonNumber           int    `json:"seasonNumber"`
		EpisodeNumbers         []int  `json:"episodeNumbers"`
		AbsoluteEpisodeNumbers []int  `json:"absoluteEpisodeNumbers"`
		Year                   int    `json:"year"`
		ReleaseGroup           string `json:"releaseGroup"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("parse golden file: %v", err)
	}

	for _, tc := range cases {
		t.Run(tc.Title, func(t *testing.T) {
			got := ParseTitle(tc.Title)
			if got.SeriesTitle != tc.SeriesTitle {
				t.Errorf("SeriesTitle = %q, want %q", got.SeriesTitle, tc.SeriesTitle)
			}
			if string(got.SeriesType) != tc.SeriesType {
				t.Errorf("SeriesType = %q, want %q", got.SeriesType, tc.SeriesType)
			}
			if got.SeasonNumber != tc.SeasonNumber {
				t.Errorf("SeasonNumber = %d, want %d", got.SeasonNumber, tc.SeasonNumber)
			}
			if !slicesEqual(got.EpisodeNumbers, tc.EpisodeNumbers) {
				t.Errorf("EpisodeNumbers = %v, want %v", got.EpisodeNumbers, tc.EpisodeNumbers)
			}
			if !slicesEqual(got.AbsoluteEpisodeNumbers, tc.AbsoluteEpisodeNumbers) {
				t.Errorf("AbsoluteEpisodeNumbers = %v, want %v", got.AbsoluteEpisodeNumbers, tc.AbsoluteEpisodeNumbers)
			}
			if got.ReleaseGroup != tc.ReleaseGroup {
				t.Errorf("ReleaseGroup = %q, want %q", got.ReleaseGroup, tc.ReleaseGroup)
			}
		})
	}
}
