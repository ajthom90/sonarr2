package organizer

import "testing"

func TestBuildFilenameDefault(t *testing.T) {
	info := EpisodeInfo{
		SeriesTitle:   "The Simpsons",
		SeasonNumber:  35,
		EpisodeNumber: 10,
		EpisodeTitle:  "Pilot",
		QualityName:   "WEBDL-1080p",
	}
	got := BuildFilename(DefaultEpisodeFormat, info)
	want := "The Simpsons - S35E10 - Pilot [WEBDL-1080p]"
	if got != want {
		t.Errorf("BuildFilename = %q, want %q", got, want)
	}
}

func TestBuildFilenameCleansIllegalChars(t *testing.T) {
	info := EpisodeInfo{
		SeriesTitle:   "Who: Wants to Be?",
		SeasonNumber:  1,
		EpisodeNumber: 1,
		EpisodeTitle:  "Pilot",
		QualityName:   "HDTV-720p",
	}
	got := BuildFilename(DefaultEpisodeFormat, info)
	// Colon and question mark should be stripped from the series title.
	for _, illegal := range []string{":", "?"} {
		if containsChar(got, illegal) {
			t.Errorf("BuildFilename result %q still contains illegal char %q", got, illegal)
		}
	}
}

func TestBuildSeasonFolder(t *testing.T) {
	tests := []struct {
		season int
		want   string
	}{
		{1, "Season 01"},
		{12, "Season 12"},
	}
	for _, tt := range tests {
		got := BuildSeasonFolder(tt.season)
		if got != tt.want {
			t.Errorf("BuildSeasonFolder(%d) = %q, want %q", tt.season, got, tt.want)
		}
	}
}

// containsChar reports whether s contains the substring sub.
func containsChar(s, sub string) bool {
	return len(s) > 0 && len(sub) > 0 && (func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
