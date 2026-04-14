package library_test

import (
	"context"
	"testing"
	"time"

	"github.com/ajthom90/sonarr2/internal/library"
)

func TestApplyMonitorMode(t *testing.T) {
	makeEpisodes := func() []library.Episode {
		past := time.Now().Add(-30 * 24 * time.Hour)
		future := time.Now().Add(30 * 24 * time.Hour)
		fileID10 := int64(10)
		fileID11 := int64(11)
		return []library.Episode{
			{ID: 1, SeriesID: 1, SeasonNumber: 1, EpisodeNumber: 1, AirDateUtc: &past, Monitored: false, EpisodeFileID: &fileID10},
			{ID: 2, SeriesID: 1, SeasonNumber: 1, EpisodeNumber: 2, AirDateUtc: &past, Monitored: false, EpisodeFileID: nil},
			{ID: 3, SeriesID: 1, SeasonNumber: 1, EpisodeNumber: 3, AirDateUtc: &past, Monitored: false, EpisodeFileID: &fileID11},
			{ID: 4, SeriesID: 1, SeasonNumber: 2, EpisodeNumber: 1, AirDateUtc: &future, Monitored: false, EpisodeFileID: nil},
			{ID: 5, SeriesID: 1, SeasonNumber: 2, EpisodeNumber: 2, AirDateUtc: &future, Monitored: false, EpisodeFileID: nil},
			{ID: 6, SeriesID: 1, SeasonNumber: 2, EpisodeNumber: 3, AirDateUtc: &future, Monitored: false, EpisodeFileID: nil},
		}
	}

	cases := []struct {
		mode     string
		wantByID map[int64]bool
	}{
		{"all", map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}},
		{"", map[int64]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}},
		{"none", map[int64]bool{1: false, 2: false, 3: false, 4: false, 5: false, 6: false}},
		{"future", map[int64]bool{1: false, 2: false, 3: false, 4: true, 5: true, 6: true}},
		{"missing", map[int64]bool{1: false, 2: true, 3: false, 4: true, 5: true, 6: true}},
		{"existing", map[int64]bool{1: true, 2: false, 3: true, 4: false, 5: false, 6: false}},
		{"pilot", map[int64]bool{1: true, 2: false, 3: false, 4: false, 5: false, 6: false}},
		{"firstSeason", map[int64]bool{1: true, 2: true, 3: true, 4: false, 5: false, 6: false}},
		{"lastSeason", map[int64]bool{1: false, 2: false, 3: false, 4: true, 5: true, 6: true}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.mode, func(t *testing.T) {
			fake := &fakeMonitorSink{eps: makeEpisodes(), updated: map[int64]bool{}}
			err := library.ApplyMonitorMode(context.Background(), fake, 1, tc.mode)
			if err != nil {
				t.Fatalf("ApplyMonitorMode: %v", err)
			}
			for id, want := range tc.wantByID {
				got, ok := fake.updated[id]
				if !ok {
					t.Errorf("episode %d: never updated", id)
					continue
				}
				if got != want {
					t.Errorf("episode %d: got monitored=%v want %v", id, got, want)
				}
			}
		})
	}
}

func TestApplyMonitorMode_UnknownModeErrors(t *testing.T) {
	fake := &fakeMonitorSink{eps: []library.Episode{{ID: 1, SeasonNumber: 1, EpisodeNumber: 1}}, updated: map[int64]bool{}}
	err := library.ApplyMonitorMode(context.Background(), fake, 1, "bogus")
	if err == nil {
		t.Fatalf("expected error for unknown mode, got nil")
	}
}

type fakeMonitorSink struct {
	eps     []library.Episode
	updated map[int64]bool
}

func (f *fakeMonitorSink) ListForSeries(ctx context.Context, seriesID int64) ([]library.Episode, error) {
	return f.eps, nil
}
func (f *fakeMonitorSink) SetMonitored(ctx context.Context, episodeID int64, monitored bool) error {
	f.updated[episodeID] = monitored
	return nil
}
