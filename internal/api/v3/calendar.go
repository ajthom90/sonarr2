package v3

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

// calendarHandler handles /api/v3/calendar endpoints.
type calendarHandler struct {
	episodes library.EpisodesStore
	log      *slog.Logger
}

// NewCalendarHandler constructs a calendarHandler.
func NewCalendarHandler(episodes library.EpisodesStore, log *slog.Logger) *calendarHandler {
	return &calendarHandler{episodes: episodes, log: log}
}

// MountCalendar registers /api/v3/calendar routes.
func MountCalendar(r chi.Router, h *calendarHandler) {
	r.Route("/api/v3/calendar", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

// list handles GET /api/v3/calendar?start=...&end=...
func (h *calendarHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	const dateFmt = "2006-01-02"

	var start, end time.Time
	var err error

	if s := q.Get("start"); s != "" {
		start, err = time.Parse(dateFmt, s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid start date, expected YYYY-MM-DD")
			return
		}
	}
	if e := q.Get("end"); e != "" {
		end, err = time.Parse(dateFmt, e)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid end date, expected YYYY-MM-DD")
			return
		}
		// Include the full end day.
		end = end.Add(24*time.Hour - time.Nanosecond)
	}

	all, err := h.episodes.ListAll(ctx)
	if err != nil {
		h.log.Error("calendar list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	var filtered []library.Episode
	for _, ep := range all {
		if ep.AirDateUtc == nil {
			continue
		}
		airDate := ep.AirDateUtc.UTC()
		if !start.IsZero() && airDate.Before(start) {
			continue
		}
		if !end.IsZero() && airDate.After(end) {
			continue
		}
		filtered = append(filtered, ep)
	}

	resources := make([]episodeResource, 0, len(filtered))
	for _, ep := range filtered {
		resources = append(resources, toEpisodeResource(ep))
	}
	writeJSON(w, http.StatusOK, resources)
}
