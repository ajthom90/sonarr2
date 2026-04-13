package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/library"
	"github.com/ajthom90/sonarr2/internal/providers/metadatasource"
)

// Invalidator is an optional interface for metadata sources that support
// cache invalidation. If the source implements it, Invalidate is called
// before fetching to ensure user-initiated refreshes bypass any cache.
type Invalidator interface {
	Invalidate(tvdbID int64)
}

// RefreshSeriesHandler fetches metadata for a series from the metadata source
// and upserts seasons and episodes into the library.
type RefreshSeriesHandler struct {
	source  metadatasource.MetadataSource
	library *library.Library
}

// NewRefreshSeriesHandler creates a RefreshSeriesHandler wired to the given
// metadata source and library.
func NewRefreshSeriesHandler(source metadatasource.MetadataSource, lib *library.Library) *RefreshSeriesHandler {
	return &RefreshSeriesHandler{source: source, library: lib}
}

// Handle implements commands.Handler. The command body must be JSON with a
// "seriesId" field: {"seriesId": 123}.
func (h *RefreshSeriesHandler) Handle(ctx context.Context, cmd commands.Command) error {
	// 1. Parse command body.
	var body struct {
		SeriesID int64 `json:"seriesId"`
	}
	if err := json.Unmarshal(cmd.Body, &body); err != nil {
		return fmt.Errorf("parse body: %w", err)
	}

	// 2. Load series from library to get TVDB ID.
	series, err := h.library.Series.Get(ctx, body.SeriesID)
	if err != nil {
		return fmt.Errorf("get series %d: %w", body.SeriesID, err)
	}

	// 2b. Invalidate cache if the source supports it.
	if inv, ok := h.source.(Invalidator); ok {
		inv.Invalidate(series.TvdbID)
	}

	// 3. Fetch metadata from source.
	info, err := h.source.GetSeries(ctx, series.TvdbID)
	if err != nil {
		return fmt.Errorf("get series metadata: %w", err)
	}

	// 4. Update series fields from metadata.
	series.Title = info.Title
	series.Status = info.Status
	if err := h.library.Series.Update(ctx, series); err != nil {
		return fmt.Errorf("update series: %w", err)
	}

	// 5. Fetch episodes.
	episodes, err := h.source.GetEpisodes(ctx, series.TvdbID)
	if err != nil {
		return fmt.Errorf("get episodes: %w", err)
	}

	// 6. Upsert seasons (distinct season numbers).
	seasonsSeen := map[int]bool{}
	for _, ep := range episodes {
		if !seasonsSeen[ep.SeasonNumber] {
			seasonsSeen[ep.SeasonNumber] = true
			h.library.Seasons.Upsert(ctx, library.Season{ //nolint:errcheck
				SeriesID:     series.ID,
				SeasonNumber: int32(ep.SeasonNumber),
				Monitored:    true,
			})
		}
	}

	// 7. Upsert episodes.
	// Load existing episodes, match by (season_number, episode_number),
	// create missing ones, update existing.
	existing, err := h.library.Episodes.ListForSeries(ctx, series.ID)
	if err != nil {
		return fmt.Errorf("list existing episodes: %w", err)
	}
	existingMap := map[string]library.Episode{}
	for _, ep := range existing {
		key := fmt.Sprintf("%d-%d", ep.SeasonNumber, ep.EpisodeNumber)
		existingMap[key] = ep
	}
	for _, ep := range episodes {
		key := fmt.Sprintf("%d-%d", ep.SeasonNumber, ep.EpisodeNumber)
		var absNum *int32
		if ep.AbsoluteEpisodeNumber != nil {
			n := int32(*ep.AbsoluteEpisodeNumber)
			absNum = &n
		}
		if existing, ok := existingMap[key]; ok {
			// Update existing episode.
			existing.Title = ep.Title
			existing.Overview = ep.Overview
			existing.AirDateUtc = ep.AirDate
			existing.AbsoluteEpisodeNumber = absNum
			h.library.Episodes.Update(ctx, existing) //nolint:errcheck
		} else {
			// Create new episode.
			h.library.Episodes.Create(ctx, library.Episode{ //nolint:errcheck
				SeriesID:              series.ID,
				SeasonNumber:          int32(ep.SeasonNumber),
				EpisodeNumber:         int32(ep.EpisodeNumber),
				AbsoluteEpisodeNumber: absNum,
				Title:                 ep.Title,
				Overview:              ep.Overview,
				AirDateUtc:            ep.AirDate,
				Monitored:             true,
			})
		}
	}

	return nil
}
