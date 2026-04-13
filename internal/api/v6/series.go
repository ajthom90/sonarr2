package v6

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// seriesResource is the v6 JSON shape for a series.
// Legacy fields tvRageId, languageProfileId, and isNetCore are omitted.
type seriesResource struct {
	ID               int64               `json:"id"`
	Title            string              `json:"title"`
	SortTitle        string              `json:"sortTitle"`
	Status           string              `json:"status"`
	Overview         string              `json:"overview"`
	Network          string              `json:"network"`
	AirTime          string              `json:"airTime"`
	Images           []any               `json:"images"`
	Seasons          []seasonResource    `json:"seasons"`
	Year             int                 `json:"year"`
	Path             string              `json:"path"`
	QualityProfileID int                 `json:"qualityProfileId"`
	SeasonFolder     bool                `json:"seasonFolder"`
	Monitored        bool                `json:"monitored"`
	Runtime          int                 `json:"runtime"`
	TvdbID           int64               `json:"tvdbId"`
	TvMazeID         int                 `json:"tvMazeId"`
	ImdbID           string              `json:"imdbId"`
	TmdbID           int                 `json:"tmdbId"`
	FirstAired       string              `json:"firstAired"`
	LastAired        string              `json:"lastAired"`
	SeriesType       string              `json:"seriesType"`
	CleanTitle       string              `json:"cleanTitle"`
	TitleSlug        string              `json:"titleSlug"`
	RootFolderPath   string              `json:"rootFolderPath"`
	Genres           []string            `json:"genres"`
	Tags             []int               `json:"tags"`
	Added            string              `json:"added"`
	Ratings          map[string]any      `json:"ratings"`
	Statistics       *statisticsResource `json:"statistics"`
	AlternateTitles  []any               `json:"alternateTitles"`
	OriginalLanguage map[string]any      `json:"originalLanguage"`
	MonitorNewItems  string              `json:"monitorNewItems"`
	Ended            bool                `json:"ended"`
}

// seasonResource is the v6 JSON shape for a season.
type seasonResource struct {
	SeasonNumber int                 `json:"seasonNumber"`
	Monitored    bool                `json:"monitored"`
	Statistics   *statisticsResource `json:"statistics,omitempty"`
}

// statisticsResource is the v6 JSON shape for series/season statistics.
type statisticsResource struct {
	SeasonCount       int     `json:"seasonCount,omitempty"`
	EpisodeFileCount  int     `json:"episodeFileCount"`
	EpisodeCount      int     `json:"episodeCount"`
	TotalEpisodeCount int     `json:"totalEpisodeCount"`
	SizeOnDisk        int64   `json:"sizeOnDisk"`
	PercentOfEpisodes float64 `json:"percentOfEpisodes"`
}

type seriesHandler struct {
	series  library.SeriesStore
	seasons library.SeasonsStore
	stats   library.SeriesStatsStore
	log     *slog.Logger
}

func newSeriesHandler(series library.SeriesStore, seasons library.SeasonsStore, stats library.SeriesStatsStore, log *slog.Logger) *seriesHandler {
	if log == nil {
		log = slog.Default()
	}
	return &seriesHandler{series: series, seasons: seasons, stats: stats, log: log}
}

func mountSeries(r chi.Router, h *seriesHandler) {
	r.Route("/series", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

// sortTitle strips leading articles and lowercases.
func sortTitle(title string) string {
	t := strings.ToLower(title)
	for _, prefix := range []string{"the ", "a ", "an "} {
		if strings.HasPrefix(t, prefix) {
			t = strings.TrimPrefix(t, prefix)
			break
		}
	}
	return strings.TrimSpace(t)
}

// cleanTitle returns a lowercase alphanumeric-only title.
func cleanTitle(title string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(title) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// formatTime formats a time.Time as RFC3339. Returns "" for zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (h *seriesHandler) toResource(s library.Series, seasons []library.Season, stats *library.SeriesStatistics) seriesResource {
	seasonResources := make([]seasonResource, 0, len(seasons))
	for _, season := range seasons {
		seasonResources = append(seasonResources, seasonResource{
			SeasonNumber: int(season.SeasonNumber),
			Monitored:    season.Monitored,
		})
	}

	var statsRes *statisticsResource
	if stats != nil {
		var pct float64
		if stats.EpisodeCount > 0 {
			pct = float64(stats.EpisodeFileCount) / float64(stats.EpisodeCount) * 100
		}
		statsRes = &statisticsResource{
			SeasonCount:       len(seasons),
			EpisodeFileCount:  int(stats.EpisodeFileCount),
			EpisodeCount:      int(stats.EpisodeCount),
			TotalEpisodeCount: int(stats.EpisodeCount),
			SizeOnDisk:        stats.SizeOnDisk,
			PercentOfEpisodes: pct,
		}
	}

	rootFolder := ""
	if s.Path != "" {
		rootFolder = filepath.Dir(s.Path)
	}

	return seriesResource{
		ID:               s.ID,
		Title:            s.Title,
		SortTitle:        sortTitle(s.Title),
		Status:           s.Status,
		Images:           []any{},
		Seasons:          seasonResources,
		Path:             s.Path,
		QualityProfileID: 1,
		SeasonFolder:     true,
		Monitored:        s.Monitored,
		TvdbID:           s.TvdbID,
		SeriesType:       s.SeriesType,
		CleanTitle:       cleanTitle(s.Title),
		TitleSlug:        s.Slug,
		RootFolderPath:   rootFolder,
		Genres:           []string{},
		Tags:             []int{},
		Added:            formatTime(s.Added),
		Ratings:          map[string]any{},
		Statistics:       statsRes,
		AlternateTitles:  []any{},
		OriginalLanguage: map[string]any{},
		MonitorNewItems:  "all",
		Ended:            s.Status == "ended",
	}
}

// list handles GET /api/v6/series with cursor pagination.
func (h *seriesHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, lastID, err := ParsePaginationParams(r)
	if err != nil {
		WriteBadRequest(w, r, err.Error())
		return
	}

	all, err := h.series.List(ctx)
	if err != nil {
		h.log.Error("v6 series list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Filter to records where id > lastID (cursor-based pagination).
	var filtered []library.Series
	for _, s := range all {
		if s.ID > lastID {
			filtered = append(filtered, s)
		}
	}

	// Take limit+1 to detect hasMore.
	hasMore := false
	if len(filtered) > limit {
		hasMore = true
		filtered = filtered[:limit]
	}

	resources := make([]seriesResource, 0, len(filtered))
	for _, s := range filtered {
		seasons, _ := h.seasons.ListForSeries(ctx, s.ID)
		stats, _ := h.stats.Get(ctx, s.ID)
		var statsPtr *library.SeriesStatistics
		if stats.SeriesID != 0 {
			statsPtr = &stats
		}
		resources = append(resources, h.toResource(s, seasons, statsPtr))
	}

	var nextCursor string
	if hasMore && len(filtered) > 0 {
		nextCursor = EncodeCursor(filtered[len(filtered)-1].ID)
	}

	writeJSON(w, http.StatusOK, Page[seriesResource]{
		Data: resources,
		Pagination: Pagination{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

// get handles GET /api/v6/series/{id}.
func (h *seriesHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	s, err := h.series.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No series with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 series get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	seasons, _ := h.seasons.ListForSeries(ctx, s.ID)
	stats, _ := h.stats.Get(ctx, s.ID)
	var statsPtr *library.SeriesStatistics
	if stats.SeriesID != 0 {
		statsPtr = &stats
	}

	writeJSON(w, http.StatusOK, h.toResource(s, seasons, statsPtr))
}

// seriesInput is the JSON body for POST and PUT requests.
type seriesInput struct {
	Title      string `json:"title"`
	TvdbID     int64  `json:"tvdbId"`
	Slug       string `json:"titleSlug"`
	Status     string `json:"status"`
	SeriesType string `json:"seriesType"`
	Path       string `json:"path"`
	Monitored  bool   `json:"monitored"`
	Seasons    []struct {
		SeasonNumber int  `json:"seasonNumber"`
		Monitored    bool `json:"monitored"`
	} `json:"seasons"`
}

// create handles POST /api/v6/series.
func (h *seriesHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input seriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	if input.Status == "" {
		input.Status = "continuing"
	}
	if input.SeriesType == "" {
		input.SeriesType = "standard"
	}
	if input.Slug == "" {
		input.Slug = strings.ToLower(strings.ReplaceAll(input.Title, " ", "-"))
	}

	s, err := h.series.Create(ctx, library.Series{
		TvdbID:     input.TvdbID,
		Title:      input.Title,
		Slug:       input.Slug,
		Status:     input.Status,
		SeriesType: input.SeriesType,
		Path:       input.Path,
		Monitored:  input.Monitored,
	})
	if err != nil {
		h.log.Error("v6 series create", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	for _, seasonIn := range input.Seasons {
		_ = h.seasons.Upsert(ctx, library.Season{
			SeriesID:     s.ID,
			SeasonNumber: int32(seasonIn.SeasonNumber),
			Monitored:    seasonIn.Monitored,
		})
	}

	seasons, _ := h.seasons.ListForSeries(ctx, s.ID)
	writeJSON(w, http.StatusCreated, h.toResource(s, seasons, nil))
}

// update handles PUT /api/v6/series/{id}.
func (h *seriesHandler) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	existing, err := h.series.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No series with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 series update get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input seriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	if input.Title != "" {
		existing.Title = input.Title
	}
	if input.Status != "" {
		existing.Status = input.Status
	}
	if input.SeriesType != "" {
		existing.SeriesType = input.SeriesType
	}
	if input.Path != "" {
		existing.Path = input.Path
	}
	if input.Slug != "" {
		existing.Slug = input.Slug
	}
	existing.Monitored = input.Monitored

	if err := h.series.Update(ctx, existing); err != nil {
		h.log.Error("v6 series update", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	for _, seasonIn := range input.Seasons {
		_ = h.seasons.Upsert(ctx, library.Season{
			SeriesID:     existing.ID,
			SeasonNumber: int32(seasonIn.SeasonNumber),
			Monitored:    seasonIn.Monitored,
		})
	}

	seasons, _ := h.seasons.ListForSeries(ctx, existing.ID)
	stats, _ := h.stats.Get(ctx, existing.ID)
	var statsPtr *library.SeriesStatistics
	if stats.SeriesID != 0 {
		statsPtr = &stats
	}

	writeJSON(w, http.StatusOK, h.toResource(existing, seasons, statsPtr))
}

// delete handles DELETE /api/v6/series/{id}.
func (h *seriesHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	if err := h.series.Delete(ctx, id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			WriteNotFound(w, r, fmt.Sprintf("No series with id %d", id))
			return
		}
		h.log.Error("v6 series delete", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
