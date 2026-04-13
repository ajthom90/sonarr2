package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/history"
)

// historyResource is the Sonarr v3 JSON shape for a history entry.
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

// pagedHistoryResponse matches Sonarr's paged envelope.
type pagedHistoryResponse struct {
	Page          int               `json:"page"`
	PageSize      int               `json:"pageSize"`
	SortKey       string            `json:"sortKey"`
	SortDirection string            `json:"sortDirection"`
	TotalRecords  int               `json:"totalRecords"`
	Records       []historyResource `json:"records"`
}

// historyHandler handles /api/v3/history endpoints.
type historyHandler struct {
	store history.Store
	log   *slog.Logger
}

// NewHistoryHandler constructs a historyHandler.
func NewHistoryHandler(store history.Store, log *slog.Logger) *historyHandler {
	return &historyHandler{store: store, log: log}
}

// MountHistory registers /api/v3/history routes.
func MountHistory(r chi.Router, h *historyHandler) {
	r.Route("/api/v3/history", func(r chi.Router) {
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

// list handles GET /api/v3/history.
func (h *historyHandler) list(w http.ResponseWriter, r *http.Request) {
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
		sortKey = "date"
	}
	sortDirection := q.Get("sortDirection")
	if sortDirection == "" {
		sortDirection = "descending"
	}

	all, err := h.store.ListAll(ctx)
	if err != nil {
		h.log.Error("history list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	total := len(all)
	offset := (page - 1) * pageSize
	if offset >= total {
		offset = total
	}
	end := offset + pageSize
	if end > total {
		end = total
	}
	pageRecords := all[offset:end]

	records := make([]historyResource, 0, len(pageRecords))
	for _, e := range pageRecords {
		records = append(records, toHistoryResource(e))
	}

	writeJSON(w, http.StatusOK, pagedHistoryResponse{
		Page:          page,
		PageSize:      pageSize,
		SortKey:       sortKey,
		SortDirection: sortDirection,
		TotalRecords:  total,
		Records:       records,
	})
}
