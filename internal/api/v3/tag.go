// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/Tags/TagController.cs,
//            src/Sonarr.Api.V3/Tags/TagResource.cs,
//            src/Sonarr.Api.V3/Tags/TagDetailsResource.cs.

package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/tags"
)

// tagResource is Sonarr's v3 JSON shape for a tag: {id, label}.
type tagResource struct {
	ID    int    `json:"id"`
	Label string `json:"label"`
}

// tagDetailsResource is the shape for /api/v3/tag/detail. Sonarr returns per-tag
// usage across delay profiles, import lists, notifications, etc. sonarr2 returns
// empty arrays for the subsystems that aren't yet wired to tags; fields must all
// be present (non-null) for clients to parse correctly.
type tagDetailsResource struct {
	ID                int   `json:"id"`
	Label             string `json:"label"`
	DelayProfileIds   []int `json:"delayProfileIds"`
	ImportListIds     []int `json:"importListIds"`
	NotificationIds   []int `json:"notificationIds"`
	RestrictionIds    []int `json:"restrictionIds"`
	IndexerIds        []int `json:"indexerIds"`
	DownloadClientIds []int `json:"downloadClientIds"`
	AutoTagIds        []int `json:"autoTagIds"`
	SeriesIds         []int `json:"seriesIds"`
}

// TagHandler handles /api/v3/tag endpoints.
type TagHandler struct {
	store tags.Store
	log   *slog.Logger
}

// NewTagHandler constructs a TagHandler.
func NewTagHandler(store tags.Store, log *slog.Logger) *TagHandler {
	return &TagHandler{store: store, log: log}
}

// MountTag registers /api/v3/tag routes. When h is nil a read-only empty stub
// is used (backwards compat for tests that don't wire a store).
func MountTag(r chi.Router, h *TagHandler) {
	if h == nil {
		r.Route("/api/v3/tag", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []any{})
			})
			r.Get("/detail", func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, http.StatusOK, []any{})
			})
		})
		return
	}
	r.Route("/api/v3/tag", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/detail", h.listDetails)
		r.Get("/detail/{id}", h.getDetails)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toTagResource(t tags.Tag) tagResource {
	return tagResource{ID: t.ID, Label: t.Label}
}

func toTagDetailsResource(t tags.Tag) tagDetailsResource {
	// Until other subsystems start using tags, all usage lists are empty.
	// They must be empty arrays, not null, to match Sonarr's wire format.
	return tagDetailsResource{
		ID:                t.ID,
		Label:             t.Label,
		DelayProfileIds:   []int{},
		ImportListIds:     []int{},
		NotificationIds:   []int{},
		RestrictionIds:    []int{},
		IndexerIds:        []int{},
		DownloadClientIds: []int{},
		AutoTagIds:        []int{},
		SeriesIds:         []int{},
	}
}

func (h *TagHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("tag list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]tagResource, 0, len(all))
	for _, t := range all {
		out = append(out, toTagResource(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *TagHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	t, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, tags.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("tag get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toTagResource(t))
}

func (h *TagHandler) create(w http.ResponseWriter, r *http.Request) {
	var body tagResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	t, err := h.store.Create(r.Context(), body.Label)
	if errors.Is(err, tags.ErrDuplicateLabel) {
		writeError(w, http.StatusBadRequest, "Label already in use")
		return
	}
	if err != nil {
		h.log.Error("tag create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toTagResource(t))
}

func (h *TagHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	var body tagResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	body.ID = id
	if err := h.store.Update(r.Context(), tags.Tag{ID: body.ID, Label: body.Label}); err != nil {
		if errors.Is(err, tags.ErrDuplicateLabel) {
			writeError(w, http.StatusBadRequest, "Label already in use")
			return
		}
		h.log.Error("tag update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// Sonarr returns 202 Accepted with the resource on PUT.
	writeJSON(w, http.StatusAccepted, toTagResource(tags.Tag{ID: body.ID, Label: tags.NormalizeLabel(body.Label)}))
}

func (h *TagHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.log.Error("tag delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TagHandler) listDetails(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("tag detail list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]tagDetailsResource, 0, len(all))
	for _, t := range all {
		out = append(out, toTagDetailsResource(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *TagHandler) getDetails(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	t, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, tags.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("tag detail get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toTagDetailsResource(t))
}
