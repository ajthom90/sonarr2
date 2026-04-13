package v6

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/library"
)

type calendarHandler struct {
	episodes library.EpisodesStore
	log      *slog.Logger
}

func newCalendarHandler(episodes library.EpisodesStore, log *slog.Logger) *calendarHandler {
	if log == nil {
		log = slog.Default()
	}
	return &calendarHandler{episodes: episodes, log: log}
}

func mountCalendar(r chi.Router, h *calendarHandler) {
	r.Route("/calendar", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

// list handles GET /api/v6/calendar?start=...&end=...
func (h *calendarHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	const dateFmt = "2006-01-02"

	var start, end time.Time
	var err error

	if s := q.Get("start"); s != "" {
		start, err = time.Parse(dateFmt, s)
		if err != nil {
			WriteBadRequest(w, r, "Invalid start date, expected YYYY-MM-DD")
			return
		}
	}
	if e := q.Get("end"); e != "" {
		end, err = time.Parse(dateFmt, e)
		if err != nil {
			WriteBadRequest(w, r, "Invalid end date, expected YYYY-MM-DD")
			return
		}
		// Include the full end day.
		end = end.Add(24*time.Hour - time.Nanosecond)
	}

	all, err := h.episodes.ListAll(ctx)
	if err != nil {
		h.log.Error("v6 calendar list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
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
