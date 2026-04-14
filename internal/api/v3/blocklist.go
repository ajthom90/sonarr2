// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/Blocklist/BlocklistController.cs,
//            src/Sonarr.Api.V3/Blocklist/BlocklistResource.cs.

package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/blocklist"
)

// blocklistResource is the Sonarr v3 JSON shape for a blocklist entry.
type blocklistResource struct {
	ID            int             `json:"id"`
	SeriesID      int             `json:"seriesId"`
	EpisodeIDs    []int           `json:"episodeIds"`
	SourceTitle   string          `json:"sourceTitle"`
	Languages     json.RawMessage `json:"languages"`
	Quality       json.RawMessage `json:"quality"`
	CustomFormats []any           `json:"customFormats"`
	Date          string          `json:"date"`
	Protocol      string          `json:"protocol"`
	Indexer       string          `json:"indexer"`
	Message       string          `json:"message"`
}

// pagedBlocklistResponse matches Sonarr's paged envelope.
type pagedBlocklistResponse struct {
	Page          int                 `json:"page"`
	PageSize      int                 `json:"pageSize"`
	SortKey       string              `json:"sortKey"`
	SortDirection string              `json:"sortDirection"`
	TotalRecords  int                 `json:"totalRecords"`
	Records       []blocklistResource `json:"records"`
}

// blocklistBulkRequest is the DELETE /api/v3/blocklist/bulk body.
type blocklistBulkRequest struct {
	Ids []int `json:"ids"`
}

// BlocklistHandler handles /api/v3/blocklist endpoints.
type BlocklistHandler struct {
	store blocklist.Store
	log   *slog.Logger
}

// NewBlocklistHandler constructs a BlocklistHandler.
func NewBlocklistHandler(store blocklist.Store, log *slog.Logger) *BlocklistHandler {
	return &BlocklistHandler{store: store, log: log}
}

// MountBlocklist registers /api/v3/blocklist routes.
func MountBlocklist(r chi.Router, h *BlocklistHandler) {
	r.Route("/api/v3/blocklist", func(r chi.Router) {
		r.Get("/", h.list)
		r.Delete("/bulk", h.bulkDelete)
		r.Delete("/{id}", h.delete)
	})
}

func toBlocklistResource(e blocklist.Entry) blocklistResource {
	langs := e.Languages
	if len(langs) == 0 {
		langs = json.RawMessage("[]")
	}
	qual := e.Quality
	if len(qual) == 0 {
		qual = json.RawMessage("{}")
	}
	return blocklistResource{
		ID:            e.ID,
		SeriesID:      e.SeriesID,
		EpisodeIDs:    nonNilIntSlice(e.EpisodeIDs),
		SourceTitle:   e.SourceTitle,
		Languages:     json.RawMessage(langs),
		Quality:       json.RawMessage(qual),
		CustomFormats: []any{},
		Date:          e.Date.UTC().Format("2006-01-02T15:04:05Z"),
		Protocol:      string(e.Protocol),
		Indexer:       e.Indexer,
		Message:       e.Message,
	}
}

func nonNilIntSlice(v []int) []int {
	if v == nil {
		return []int{}
	}
	return v
}

func (h *BlocklistHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize < 1 {
		pageSize = 20
	}
	sortKey := q.Get("sortKey")
	if sortKey == "" {
		sortKey = "date"
	}
	sortDir := q.Get("sortDirection")
	if sortDir == "" {
		sortDir = "descending"
	}

	pg, err := h.store.List(r.Context(), page, pageSize)
	if err != nil {
		h.log.Error("blocklist list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}

	records := make([]blocklistResource, 0, len(pg.Records))
	for _, e := range pg.Records {
		records = append(records, toBlocklistResource(e))
	}

	writeJSON(w, http.StatusOK, pagedBlocklistResponse{
		Page:          pg.Page,
		PageSize:      pg.PageSize,
		SortKey:       sortKey,
		SortDirection: sortDir,
		TotalRecords:  pg.TotalRecords,
		Records:       records,
	})
}

func (h *BlocklistHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, blocklist.ErrNotFound) {
			writeError(w, http.StatusNotFound, "Not Found")
			return
		}
		h.log.Error("blocklist delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *BlocklistHandler) bulkDelete(w http.ResponseWriter, r *http.Request) {
	var body blocklistBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if err := h.store.DeleteMany(r.Context(), body.Ids); err != nil {
		h.log.Error("blocklist bulk delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// Sonarr returns `{}` for bulk delete.
	writeJSON(w, http.StatusOK, map[string]any{})
}
