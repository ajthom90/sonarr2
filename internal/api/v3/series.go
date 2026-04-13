// Package v3 implements the Sonarr v3 REST API surface.
package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// seriesResource is the Sonarr v3 JSON shape for a series.
type seriesResource struct {
	ID                int64               `json:"id"`
	Title             string              `json:"title"`
	SortTitle         string              `json:"sortTitle"`
	Status            string              `json:"status"`
	Overview          string              `json:"overview"`
	Network           string              `json:"network"`
	AirTime           string              `json:"airTime"`
	Images            []any               `json:"images"`
	Seasons           []seasonResource    `json:"seasons"`
	Year              int                 `json:"year"`
	Path              string              `json:"path"`
	QualityProfileID  int                 `json:"qualityProfileId"`
	SeasonFolder      bool                `json:"seasonFolder"`
	Monitored         bool                `json:"monitored"`
	Runtime           int                 `json:"runtime"`
	TvdbID            int64               `json:"tvdbId"`
	TvRageID          int                 `json:"tvRageId"`
	TvMazeID          int                 `json:"tvMazeId"`
	ImdbID            string              `json:"imdbId"`
	TmdbID            int                 `json:"tmdbId"`
	FirstAired        string              `json:"firstAired"`
	LastAired         string              `json:"lastAired"`
	SeriesType        string              `json:"seriesType"`
	CleanTitle        string              `json:"cleanTitle"`
	TitleSlug         string              `json:"titleSlug"`
	RootFolderPath    string              `json:"rootFolderPath"`
	Genres            []string            `json:"genres"`
	Tags              []int               `json:"tags"`
	Added             string              `json:"added"`
	Ratings           map[string]any      `json:"ratings"`
	Statistics        *statisticsResource `json:"statistics"`
	AlternateTitles   []any               `json:"alternateTitles"`
	OriginalLanguage  map[string]any      `json:"originalLanguage"`
	UseSceneNumbering bool                `json:"useSceneNumbering"`
	MonitorNewItems   string              `json:"monitorNewItems"`
	Ended             bool                `json:"ended"`
}

// seasonResource is the Sonarr v3 JSON shape for a season.
type seasonResource struct {
	SeasonNumber int                 `json:"seasonNumber"`
	Monitored    bool                `json:"monitored"`
	Statistics   *statisticsResource `json:"statistics,omitempty"`
}

// statisticsResource is the Sonarr v3 JSON shape for series/season statistics.
type statisticsResource struct {
	SeasonCount       int     `json:"seasonCount,omitempty"`
	EpisodeFileCount  int     `json:"episodeFileCount"`
	EpisodeCount      int     `json:"episodeCount"`
	TotalEpisodeCount int     `json:"totalEpisodeCount"`
	SizeOnDisk        int64   `json:"sizeOnDisk"`
	PercentOfEpisodes float64 `json:"percentOfEpisodes"`
}

// seriesHandler handles all /api/v3/series endpoints.
type seriesHandler struct {
	series  library.SeriesStore
	seasons library.SeasonsStore
	stats   library.SeriesStatsStore
	log     *slog.Logger
}

// NewSeriesHandler constructs a seriesHandler.
func NewSeriesHandler(series library.SeriesStore, seasons library.SeasonsStore, stats library.SeriesStatsStore, log *slog.Logger) *seriesHandler {
	return &seriesHandler{
		series:  series,
		seasons: seasons,
		stats:   stats,
		log:     log,
	}
}

// MountSeries registers the /api/v3/series routes on r.
func MountSeries(r chi.Router, h *seriesHandler) {
	r.Route("/api/v3/series", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

// sortTitle returns a Sonarr-style sort title: lowercase with leading articles
// ("the ", "a ", "an ") stripped.
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

// cleanTitle returns a Sonarr-style clean title: lowercase, alphanumeric only.
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

// toResource converts a domain Series + seasons + stats into the Sonarr JSON shape.
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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"message":"` + message + `"}`))
}

func parseIDParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
}

// list handles GET /api/v3/series.
func (h *seriesHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	all, err := h.series.List(ctx)
	if err != nil {
		h.log.Error("series list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	resources := make([]seriesResource, 0, len(all))
	for _, s := range all {
		seasons, _ := h.seasons.ListForSeries(ctx, s.ID)
		stats, _ := h.stats.Get(ctx, s.ID)
		var statsPtr *library.SeriesStatistics
		if stats.SeriesID != 0 {
			statsPtr = &stats
		}
		resources = append(resources, h.toResource(s, seasons, statsPtr))
	}

	writeJSON(w, http.StatusOK, resources)
}

// get handles GET /api/v3/series/{id}.
func (h *seriesHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	s, err := h.series.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("series get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
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

// create handles POST /api/v3/series.
func (h *seriesHandler) create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input seriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
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
		h.log.Error("series create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Upsert any seasons provided in the request body.
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

// update handles PUT /api/v3/series/{id}.
func (h *seriesHandler) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	existing, err := h.series.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("series update get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input seriesInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Apply updates — only overwrite non-zero fields.
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
		h.log.Error("series update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Update seasons if provided.
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

// delete handles DELETE /api/v3/series/{id}.
func (h *seriesHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if err := h.series.Delete(ctx, id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("series delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
