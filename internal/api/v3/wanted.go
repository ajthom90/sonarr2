package v3

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// pagedEpisodeResponse matches Sonarr's paged envelope for episodes.
type pagedEpisodeResponse struct {
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	SortKey       string            `json:"sortKey"`
	SortDirection string            `json:"sortDirection"`
	TotalRecords  int               `json:"totalRecords"`
	Records       []episodeResource `json:"records"`
}

// wantedHandler handles /api/v3/wanted endpoints.
type wantedHandler struct {
	episodes library.EpisodesStore
	log      *slog.Logger
}

// NewWantedHandler constructs a wantedHandler.
func NewWantedHandler(episodes library.EpisodesStore, log *slog.Logger) *wantedHandler {
	return &wantedHandler{episodes: episodes, log: log}
}

// MountWanted registers /api/v3/wanted routes.
func MountWanted(r chi.Router, h *wantedHandler) {
	r.Route("/api/v3/wanted", func(r chi.Router) {
		r.Get("/missing", h.missing)
		r.Get("/cutoff", h.cutoffUnmet)
	})
}

// missing handles GET /api/v3/wanted/missing.
func (h *wantedHandler) missing(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	page := 1
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	pageSize := 20
	if ps := q.Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	sortKey := q.Get("sortKey")
	if sortKey == "" {
		sortKey = "airDateUtc"
	}
	sortDirection := q.Get("sortDirection")
	if sortDirection == "" {
		sortDirection = "descending"
	}

	all, err := h.episodes.ListAll(ctx)
	if err != nil {
		h.log.Error("wanted missing", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Filter: monitored episodes with no episode file.
	var missing []library.Episode
	for _, ep := range all {
		if ep.Monitored && ep.EpisodeFileID == nil {
			missing = append(missing, ep)
		}
	}

	total := len(missing)
	offset := (page - 1) * pageSize
	if offset >= total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	pageRecords := missing[offset:end]

	records := make([]episodeResource, 0, len(pageRecords))
	for _, ep := range pageRecords {
		records = append(records, toEpisodeResource(ep))
	}

	writeJSON(w, http.StatusOK, pagedEpisodeResponse{
		Page:          page,
		PageSize:      pageSize,
		SortKey:       sortKey,
		SortDirection: sortDirection,
		TotalRecords:  total,
		Records:       records,
	})
}

// cutoffUnmet handles GET /api/v3/wanted/cutoff. Sonarr's "cutoff unmet"
// concept lists episodes that have a file but below the series' quality
// profile's cutoff, i.e. candidates for upgrade. Computing the real upgrade
// decision requires the quality profile and decision engine; for an
// operational table we approximate as "monitored episodes that have a file."
// The decision engine independently drives actual upgrade grabs; this
// endpoint is purely informational. Tighter cutoff logic is a follow-up.
func (h *wantedHandler) cutoffUnmet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	page := 1
	if p := q.Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}

	pageSize := 20
	if ps := q.Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	sortKey := q.Get("sortKey")
	if sortKey == "" {
		sortKey = "airDateUtc"
	}
	sortDirection := q.Get("sortDirection")
	if sortDirection == "" {
		sortDirection = "descending"
	}

	all, err := h.episodes.ListAll(ctx)
	if err != nil {
		h.log.Error("wanted cutoff", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Filter: monitored episodes with a file. See doc comment above for
	// scope note on why this approximation is acceptable for the UI.
	var candidates []library.Episode
	for _, ep := range all {
		if ep.Monitored && ep.EpisodeFileID != nil {
			candidates = append(candidates, ep)
		}
	}

	total := len(candidates)
	offset := (page - 1) * pageSize
	if offset >= total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	pageRecords := candidates[offset:end]

	records := make([]episodeResource, 0, len(pageRecords))
	for _, ep := range pageRecords {
		records = append(records, toEpisodeResource(ep))
	}

	writeJSON(w, http.StatusOK, pagedEpisodeResponse{
		Page:          page,
		PageSize:      pageSize,
		SortKey:       sortKey,
		SortDirection: sortDirection,
		TotalRecords:  total,
		Records:       records,
	})
}
