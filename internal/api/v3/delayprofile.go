// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/Profiles/Delay/DelayProfileController.cs.

package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/delayprofile"
)

type delayProfileResource struct {
	ID                             int    `json:"id"`
	EnableUsenet                   bool   `json:"enableUsenet"`
	EnableTorrent                  bool   `json:"enableTorrent"`
	PreferredProtocol              string `json:"preferredProtocol"`
	UsenetDelay                    int    `json:"usenetDelay"`
	TorrentDelay                   int    `json:"torrentDelay"`
	Order                          int    `json:"order"`
	BypassIfHighestQuality         bool   `json:"bypassIfHighestQuality"`
	BypassIfAboveCustomFormatScore bool   `json:"bypassIfAboveCustomFormatScore"`
	MinimumCustomFormatScore       int    `json:"minimumCustomFormatScore"`
	Tags                           []int  `json:"tags"`
}

type DelayProfileHandler struct {
	store delayprofile.Store
	log   *slog.Logger
}

func NewDelayProfileHandler(store delayprofile.Store, log *slog.Logger) *DelayProfileHandler {
	return &DelayProfileHandler{store: store, log: log}
}

func MountDelayProfile(r chi.Router, h *DelayProfileHandler) {
	r.Route("/api/v3/delayprofile", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toDelayProfileResource(p delayprofile.Profile) delayProfileResource {
	return delayProfileResource{
		ID:                             p.ID,
		EnableUsenet:                   p.EnableUsenet,
		EnableTorrent:                  p.EnableTorrent,
		PreferredProtocol:              string(p.PreferredProtocol),
		UsenetDelay:                    p.UsenetDelay,
		TorrentDelay:                   p.TorrentDelay,
		Order:                          p.Order,
		BypassIfHighestQuality:         p.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore: p.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:       p.MinimumCustomFormatScore,
		Tags:                           nonNilIntSlice(p.Tags),
	}
}

func fromDelayProfileResource(body delayProfileResource) delayprofile.Profile {
	return delayprofile.Profile{
		ID:                             body.ID,
		EnableUsenet:                   body.EnableUsenet,
		EnableTorrent:                  body.EnableTorrent,
		PreferredProtocol:              delayprofile.Protocol(body.PreferredProtocol),
		UsenetDelay:                    body.UsenetDelay,
		TorrentDelay:                   body.TorrentDelay,
		Order:                          body.Order,
		BypassIfHighestQuality:         body.BypassIfHighestQuality,
		BypassIfAboveCustomFormatScore: body.BypassIfAboveCustomFormatScore,
		MinimumCustomFormatScore:       body.MinimumCustomFormatScore,
		Tags:                           body.Tags,
	}
}

func (h *DelayProfileHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("delayprofile list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]delayProfileResource, 0, len(all))
	for _, p := range all {
		out = append(out, toDelayProfileResource(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *DelayProfileHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	p, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, delayprofile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("delayprofile get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toDelayProfileResource(p))
}

func (h *DelayProfileHandler) create(w http.ResponseWriter, r *http.Request) {
	var body delayProfileResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	p, err := h.store.Create(r.Context(), fromDelayProfileResource(body))
	if err != nil {
		h.log.Error("delayprofile create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toDelayProfileResource(p))
}

func (h *DelayProfileHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	var body delayProfileResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	body.ID = id
	if err := h.store.Update(r.Context(), fromDelayProfileResource(body)); err != nil {
		h.log.Error("delayprofile update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusAccepted, body)
}

func (h *DelayProfileHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.log.Error("delayprofile delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
