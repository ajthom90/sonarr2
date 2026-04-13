package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// episodeResource is the Sonarr v3 JSON shape for an episode.
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

// episodeHandler handles all /api/v3/episode endpoints.
type episodeHandler struct {
	episodes library.EpisodesStore
	log      *slog.Logger
}

// NewEpisodeHandler constructs an episodeHandler.
func NewEpisodeHandler(episodes library.EpisodesStore, log *slog.Logger) *episodeHandler {
	return &episodeHandler{episodes: episodes, log: log}
}

// MountEpisode registers the /api/v3/episode routes on r.
func MountEpisode(r chi.Router, h *episodeHandler) {
	r.Route("/api/v3/episode", func(r chi.Router) {
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

// list handles GET /api/v3/episode?seriesId=N.
func (h *episodeHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	seriesIDStr := r.URL.Query().Get("seriesId")
	if seriesIDStr == "" {
		writeError(w, http.StatusBadRequest, "seriesId query parameter is required")
		return
	}
	seriesID, err := strconv.ParseInt(seriesIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid seriesId")
		return
	}

	episodes, err := h.episodes.ListForSeries(ctx, seriesID)
	if err != nil {
		h.log.Error("episode list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	resources := make([]episodeResource, 0, len(episodes))
	for _, e := range episodes {
		resources = append(resources, toEpisodeResource(e))
	}
	writeJSON(w, http.StatusOK, resources)
}

// get handles GET /api/v3/episode/{id}.
func (h *episodeHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	e, err := h.episodes.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("episode get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	writeJSON(w, http.StatusOK, toEpisodeResource(e))
}

// episodeUpdateInput is the JSON body for PUT /api/v3/episode/{id}.
type episodeUpdateInput struct {
	Monitored bool `json:"monitored"`
}

// update handles PUT /api/v3/episode/{id}.
func (h *episodeHandler) update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	existing, err := h.episodes.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("episode update get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var input episodeUpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	existing.Monitored = input.Monitored
	if err := h.episodes.Update(ctx, existing); err != nil {
		h.log.Error("episode update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	writeJSON(w, http.StatusOK, toEpisodeResource(existing))
}
