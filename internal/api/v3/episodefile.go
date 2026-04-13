package v3

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// episodeFileResource is the Sonarr v3 JSON shape for an episode file.
type episodeFileResource struct {
	ID           int64                    `json:"id"`
	SeriesID     int64                    `json:"seriesId"`
	SeasonNumber int32                    `json:"seasonNumber"`
	RelativePath string                   `json:"relativePath"`
	Path         string                   `json:"path"`
	Size         int64                    `json:"size"`
	DateAdded    string                   `json:"dateAdded"`
	Quality      episodeFileQualityHolder `json:"quality"`
	ReleaseGroup string                   `json:"releaseGroup"`
}

// episodeFileQualityHolder matches the nested quality structure Sonarr returns.
type episodeFileQualityHolder struct {
	Quality episodeFileQualityInner `json:"quality"`
}

// episodeFileQualityInner is the innermost quality object.
type episodeFileQualityInner struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	Source     string `json:"source"`
	Resolution int    `json:"resolution"`
}

// episodeFileHandler handles all /api/v3/episodefile endpoints.
type episodeFileHandler struct {
	episodeFiles library.EpisodeFilesStore
	series       library.SeriesStore
	log          *slog.Logger
}

// NewEpisodeFileHandler constructs an episodeFileHandler.
func NewEpisodeFileHandler(episodeFiles library.EpisodeFilesStore, series library.SeriesStore, log *slog.Logger) *episodeFileHandler {
	return &episodeFileHandler{episodeFiles: episodeFiles, series: series, log: log}
}

// MountEpisodeFile registers the /api/v3/episodefile routes on r.
func MountEpisodeFile(r chi.Router, h *episodeFileHandler) {
	r.Route("/api/v3/episodefile", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Delete("/{id}", h.delete)
	})
}

func (h *episodeFileHandler) toResource(f library.EpisodeFile, seriesPath string) episodeFileResource {
	fullPath := seriesPath + "/" + f.RelativePath

	return episodeFileResource{
		ID:           f.ID,
		SeriesID:     f.SeriesID,
		SeasonNumber: f.SeasonNumber,
		RelativePath: f.RelativePath,
		Path:         fullPath,
		Size:         f.Size,
		DateAdded:    formatTime(f.DateAdded),
		Quality: episodeFileQualityHolder{
			Quality: episodeFileQualityInner{
				ID:         0,
				Name:       f.QualityName,
				Source:     "",
				Resolution: 0,
			},
		},
		ReleaseGroup: f.ReleaseGroup,
	}
}

// list handles GET /api/v3/episodefile?seriesId=N.
func (h *episodeFileHandler) list(w http.ResponseWriter, r *http.Request) {
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

	files, err := h.episodeFiles.ListForSeries(ctx, seriesID)
	if err != nil {
		h.log.Error("episodefile list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Look up series path once.
	seriesPath := ""
	if len(files) > 0 {
		s, serr := h.series.Get(ctx, seriesID)
		if serr == nil {
			seriesPath = s.Path
		}
	}

	resources := make([]episodeFileResource, 0, len(files))
	for _, f := range files {
		resources = append(resources, h.toResource(f, seriesPath))
	}
	writeJSON(w, http.StatusOK, resources)
}

// get handles GET /api/v3/episodefile/{id}.
func (h *episodeFileHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	f, err := h.episodeFiles.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("episodefile get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	seriesPath := ""
	s, serr := h.series.Get(ctx, f.SeriesID)
	if serr == nil {
		seriesPath = s.Path
	}

	writeJSON(w, http.StatusOK, h.toResource(f, seriesPath))
}

// delete handles DELETE /api/v3/episodefile/{id}.
func (h *episodeFileHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if err := h.episodeFiles.Delete(ctx, id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("episodefile delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
