package library

import (
	"context"
	"fmt"
	"time"
)

// MonitorSink is the subset of EpisodesStore that ApplyMonitorMode needs.
// Narrow interface so callers can mock easily.
type MonitorSink interface {
	ListForSeries(ctx context.Context, seriesID int64) ([]Episode, error)
	SetMonitored(ctx context.Context, episodeID int64, monitored bool) error
}

// ApplyMonitorMode iterates all episodes of seriesID and sets each episode's
// monitored flag according to mode. Unknown modes return an error.
//
// Known modes:
//
//	all         - every episode monitored (default when mode == "")
//	none        - no episode monitored
//	future      - episodes with airDate > now
//	missing     - episodes without an associated file
//	existing    - episodes with an associated file
//	pilot       - only S01E01
//	firstSeason - only season 1
//	lastSeason  - only the highest season number
func ApplyMonitorMode(ctx context.Context, sink MonitorSink, seriesID int64, mode string) error {
	eps, err := sink.ListForSeries(ctx, seriesID)
	if err != nil {
		return fmt.Errorf("library: list episodes for monitor apply: %w", err)
	}
	if len(eps) == 0 {
		return nil
	}

	var maxSeason int32
	for _, e := range eps {
		if e.SeasonNumber > maxSeason {
			maxSeason = e.SeasonNumber
		}
	}

	now := time.Now()
	decide := func(e Episode) (bool, error) {
		switch mode {
		case "", "all":
			return true, nil
		case "none":
			return false, nil
		case "future":
			return e.AirDateUtc != nil && e.AirDateUtc.After(now), nil
		case "missing":
			return e.EpisodeFileID == nil, nil
		case "existing":
			return e.EpisodeFileID != nil, nil
		case "pilot":
			return e.SeasonNumber == 1 && e.EpisodeNumber == 1, nil
		case "firstSeason":
			return e.SeasonNumber == 1, nil
		case "lastSeason":
			return e.SeasonNumber == maxSeason, nil
		default:
			return false, fmt.Errorf("library: unknown monitor mode %q", mode)
		}
	}

	for _, e := range eps {
		want, err := decide(e)
		if err != nil {
			return err
		}
		if err := sink.SetMonitored(ctx, e.ID, want); err != nil {
			return fmt.Errorf("library: set monitored for episode %d: %w", e.ID, err)
		}
	}
	return nil
}
