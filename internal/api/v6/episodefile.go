package v6

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// episodeFileResource is the v6 JSON shape for an episode file.
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

// episodeFileQualityHolder matches the nested quality structure.
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

type episodeFileHandler struct {
	episodeFiles library.EpisodeFilesStore
	series       library.SeriesStore
	log          *slog.Logger
}

func newEpisodeFileHandler(episodeFiles library.EpisodeFilesStore, series library.SeriesStore, log *slog.Logger) *episodeFileHandler {
	if log == nil {
		log = slog.Default()
	}
	return &episodeFileHandler{episodeFiles: episodeFiles, series: series, log: log}
}

func mountEpisodeFile(r chi.Router, h *episodeFileHandler) {
	r.Route("/episodefile", func(r chi.Router) {
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

// list handles GET /api/v6/episodefile?seriesId=N.
func (h *episodeFileHandler) list(w http.ResponseWriter, r *http.Request) {
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

	files, err := h.episodeFiles.ListForSeries(ctx, seriesID)
	if err != nil {
		h.log.Error("v6 episodefile list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

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

// get handles GET /api/v6/episodefile/{id}.
func (h *episodeFileHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	f, err := h.episodeFiles.Get(ctx, id)
	if errors.Is(err, library.ErrNotFound) {
		WriteNotFound(w, r, fmt.Sprintf("No episode file with id %d", id))
		return
	}
	if err != nil {
		h.log.Error("v6 episodefile get", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	seriesPath := ""
	s, serr := h.series.Get(ctx, f.SeriesID)
	if serr == nil {
		seriesPath = s.Path
	}

	writeJSON(w, http.StatusOK, h.toResource(f, seriesPath))
}

// delete handles DELETE /api/v6/episodefile/{id}.
func (h *episodeFileHandler) delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := parseIDFromRequest(r)
	if err != nil {
		WriteBadRequest(w, r, "Invalid id")
		return
	}

	if err := h.episodeFiles.Delete(ctx, id); err != nil {
		if errors.Is(err, library.ErrNotFound) {
			WriteNotFound(w, r, fmt.Sprintf("No episode file with id %d", id))
			return
		}
		h.log.Error("v6 episodefile delete", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	w.WriteHeader(http.StatusOK)
}
