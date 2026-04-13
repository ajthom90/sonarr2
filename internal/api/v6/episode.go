package v6

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// episodeResource is the v6 JSON shape for an episode.
type episodeResource struct {
	ID                       int64  `json:"id"`
	SeriesID                 int64  `json:"seriesId"`
	TvdbID                   int    `json:"tvdbId"`
	EpisodeFileID            int64  `json:"episodeFileId"`
	SeasonNumber             int    `json:"seasonNumber"`
	EpisodeNumber            int    `json:"episodeNumber"`
	AbsoluteEpisodeNumber    *int   `json:"absoluteEpisodeNumber"`
	Title                    string `json:"title"`
	AirDate                  string `json:"airDate"`
	AirDateUtc               string `json:"airDateUtc"`
	Overview                 string `json:"overview"`
	HasFile                  bool   `json:"hasFile"`
	Monitored                bool   `json:"monitored"`
	Runtime                  int    `json:"runtime"`
	UnverifiedSceneNumbering bool   `json:"unverifiedSceneNumbering"`
}

type episodeHandler struct {
	episodes library.EpisodesStore
	log      *slog.Logger
}

func newEpisodeHandler(episodes library.EpisodesStore, log *slog.Logger) *episodeHandler {
	if log == nil {
		log = slog.Default()
	}
	return &episodeHandler{episodes: episodes, log: log}
}

func mountEpisode(r chi.Router, h *episodeHandler) {
	r.Route("/episode", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
	})
}

func toEpisodeResource(e library.Episode) episodeResource {
	var fileID int64
	hasFile := e.EpisodeFileID != nil
	if hasFile {
		fileID = *e.EpisodeFileID
	}

	airDate := ""
	airDateUtc := ""
	if e.AirDateUtc != nil {
		airDate = e.AirDateUtc.UTC().Format("2006-01-02")
		airDateUtc = e.AirDateUtc.UTC().Format("2006-01-02T15:04:05Z")
	}

	var absEpNum *int
	if e.AbsoluteEpisodeNumber != nil {
		n := int(*e.AbsoluteEpisodeNumber)
		absEpNum = &n
	}

	return episodeResource{
		ID:                       e.ID,
		SeriesID:                 e.SeriesID,
		TvdbID:                   0,
		EpisodeFileID:            fileID,
		SeasonNumber:             int(e.SeasonNumber),
		EpisodeNumber:            int(e.EpisodeNumber),
		AbsoluteEpisodeNumber:    absEpNum,
		Title:                    e.Title,
		AirDate:                  airDate,
		AirDateUtc:               airDateUtc,
		Overview:                 e.Overview,
		HasFile:                  hasFile,
		Monitored:                e.Monitored,
		Runtime:                  0,
		UnverifiedSceneNumbering: false,
	}
}

// list handles GET /api/v6/episode?seriesId=N with cursor pagination.
func (h *episodeHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	seriesIDStr := r.URL.Query().Get("seriesId")
	if seriesIDStr == "" {
		WriteBadRequest(w, r, "seriesId query parameter is required")
		return
	}
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		WriteBadRequest(w, r, "Invalid seriesId")
		return
	}

	limit, lastID, err := ParsePaginationParams(r)
	if err != nil {
		WriteBadRequest(w, r, err.Error())
		return
	}

	episodes, err := h.episodes.ListForSeries(ctx, seriesID)
	if err != nil {
		h.log.Error("v6 episode list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Filter to records where id > lastID.
	var filtered []library.Episode
	for _, e := range episodes {
		if e.ID > lastID {
			filtered = append(filtered, e)
		}
	}

	hasMore := false
	if len(filtered) > limit {
		hasMore = true
		filtered = filtered[:limit]
	}

	resources := make([]episodeResource, 0, len(filtered))
	for _, e := range filtered {
		resources = append(resources, toEpisodeResource(e))
	}

	var nextCursor string
	if hasMore && len(filtered) > 0 {
		nextCursor = EncodeCursor(filtered[len(filtered)-1].ID)
	}

	writeJSON(w, http.StatusOK, Page[episodeResource]{
		Data: resources,
		Pagination: Pagination{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}

// get handles GET /api/v6/episode/{id}.
func (h *episodeHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	e, err := h.episodes.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No episode with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 episode get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	writeJSON(w, http.StatusOK, toEpisodeResource(e))
}

// episodeUpdateInput is the JSON body for PUT /api/v6/episode/{id}.
type episodeUpdateInput struct {
	Monitored bool `json:"monitored"`
}

// update handles PUT /api/v6/episode/{id}.
func (h *episodeHandler) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	existing, err := h.episodes.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No episode with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 episode update get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input episodeUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteBadRequest(w, r, "Invalid request body")
		return
	}

	existing.Monitored = input.Monitored
	if err := h.episodes.Update(ctx, existing); err != nil {
		h.log.Error("v6 episode update", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	writeJSON(w, http.StatusOK, toEpisodeResource(existing))
}
