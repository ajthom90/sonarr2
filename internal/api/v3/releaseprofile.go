// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/Profiles/Release/ReleaseProfileController.cs.

package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/releaseprofile"
)

type releaseProfileResource struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	Enabled   bool     `json:"enabled"`
	Required  []string `json:"required"`
	Ignored   []string `json:"ignored"`
	IndexerID int      `json:"indexerId"`
	Tags      []int    `json:"tags"`
}

// ReleaseProfileHandler handles /api/v3/releaseprofile endpoints.
type ReleaseProfileHandler struct {
	store releaseprofile.Store
	log   *slog.Logger
}

func NewReleaseProfileHandler(store releaseprofile.Store, log *slog.Logger) *ReleaseProfileHandler {
	return &ReleaseProfileHandler{store: store, log: log}
}

func MountReleaseProfile(r chi.Router, h *ReleaseProfileHandler) {
	r.Route("/api/v3/releaseprofile", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toReleaseProfileResource(p releaseprofile.Profile) releaseProfileResource {
	return releaseProfileResource{
		ID:        p.ID,
		Name:      p.Name,
		Enabled:   p.Enabled,
		Required:  nonNilStrings(p.Required),
		Ignored:   nonNilStrings(p.Ignored),
		IndexerID: p.IndexerID,
		Tags:      nonNilIntSlice(p.Tags),
	}
}

func nonNilStrings(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}

func (h *ReleaseProfileHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("releaseprofile list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]releaseProfileResource, 0, len(all))
	for _, p := range all {
		out = append(out, toReleaseProfileResource(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ReleaseProfileHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	p, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, releaseprofile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("releaseprofile get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toReleaseProfileResource(p))
}

func (h *ReleaseProfileHandler) create(w http.ResponseWriter, r *http.Request) {
	var body releaseProfileResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	p, err := h.store.Create(r.Context(), releaseprofile.Profile{
		Name:      body.Name,
		Enabled:   body.Enabled,
		Required:  body.Required,
		Ignored:   body.Ignored,
		IndexerID: body.IndexerID,
		Tags:      body.Tags,
	})
	if err != nil {
		h.log.Error("releaseprofile create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toReleaseProfileResource(p))
}

func (h *ReleaseProfileHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	var body releaseProfileResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	body.ID = id
	if err := h.store.Update(r.Context(), releaseprofile.Profile{
		ID: id, Name: body.Name, Enabled: body.Enabled,
		Required: body.Required, Ignored: body.Ignored,
		IndexerID: body.IndexerID, Tags: body.Tags,
	}); err != nil {
		h.log.Error("releaseprofile update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusAccepted, body)
}

func (h *ReleaseProfileHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.log.Error("releaseprofile delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
