package v6

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/history"
)

// historyResource is the v6 JSON shape for a history entry.
type historyResource struct {
	ID          int64           `json:"id"`
	EpisodeID   int64           `json:"episodeId"`
	SeriesID    int64           `json:"seriesId"`
	SourceTitle string          `json:"sourceTitle"`
	QualityName string          `json:"quality"`
	EventType   string          `json:"eventType"`
	Date        string          `json:"date"`
	DownloadID  string          `json:"downloadId"`
	Data        json.RawMessage `json:"data"`
}

type historyHandler struct {
	store history.Store
	log   *slog.Logger
}

func newHistoryHandler(store history.Store, log *slog.Logger) *historyHandler {
	if log == nil {
		log = slog.Default()
	}
	return &historyHandler{store: store, log: log}
}

func mountHistory(r chi.Router, h *historyHandler) {
	r.Route("/history", func(r chi.Router) {
		r.Get("/", h.list)
	})
}

func toHistoryResource(e history.Entry) historyResource {
	data := e.Data
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	return historyResource{
		ID:          e.ID,
		EpisodeID:   e.EpisodeID,
		SeriesID:    e.SeriesID,
		SourceTitle: e.SourceTitle,
		QualityName: e.QualityName,
		EventType:   string(e.EventType),
		Date:        formatTime(e.Date),
		DownloadID:  e.DownloadID,
		Data:        data,
	}
}

// list handles GET /api/v6/history with cursor pagination.
func (h *historyHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit, lastID, err := ParsePaginationParams(r)
	if err != nil {
		WriteBadRequest(w, r, err.Error())
		return
	}

	all, err := h.store.ListAll(ctx)
	if err != nil {
		h.log.Error("v6 history list", slog.String("err", err.Error()))
		WriteError(w, r, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	// Filter to records where id > lastID.
	var filtered []history.Entry
	for _, e := range all {
		if e.ID > lastID {
			filtered = append(filtered, e)
		}
	}

	hasMore := false
	if len(filtered) > limit {
		hasMore = true
		filtered = filtered[:limit]
	}

	resources := make([]historyResource, 0, len(filtered))
	for _, e := range filtered {
		resources = append(resources, toHistoryResource(e))
	}

	var nextCursor string
	if hasMore && len(filtered) > 0 {
		nextCursor = EncodeCursor(filtered[len(filtered)-1].ID)
	}

	writeJSON(w, http.StatusOK, Page[historyResource]{
		Data: resources,
		Pagination: Pagination{
			Limit:      limit,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
	})
}
