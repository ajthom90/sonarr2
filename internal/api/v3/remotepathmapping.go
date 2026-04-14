// SPDX-License-Identifier: GPL-3.0-or-later
// Ported from Sonarr (https://github.com/Sonarr/Sonarr),
// Copyright (c) Team Sonarr, licensed under GPL-3.0.
// Reference: src/Sonarr.Api.V3/RemotePathMappings/RemotePathMappingController.cs.

package v3

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/remotepathmapping"
)

// remotePathMappingResource matches Sonarr's RemotePathMappingResource.
type remotePathMappingResource struct {
	ID         int    `json:"id"`
	Host       string `json:"host"`
	RemotePath string `json:"remotePath"`
	LocalPath  string `json:"localPath"`
}

// RemotePathMappingHandler handles /api/v3/remotepathmapping endpoints.
type RemotePathMappingHandler struct {
	store remotepathmapping.Store
	log   *slog.Logger
}

// NewRemotePathMappingHandler constructs a RemotePathMappingHandler.
func NewRemotePathMappingHandler(store remotepathmapping.Store, log *slog.Logger) *RemotePathMappingHandler {
	return &RemotePathMappingHandler{store: store, log: log}
}

// MountRemotePathMapping registers /api/v3/remotepathmapping routes.
func MountRemotePathMapping(r chi.Router, h *RemotePathMappingHandler) {
	r.Route("/api/v3/remotepathmapping", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
	})
}

func toRemotePathMappingResource(m remotepathmapping.Mapping) remotePathMappingResource {
	return remotePathMappingResource{
		ID: m.ID, Host: m.Host, RemotePath: m.RemotePath, LocalPath: m.LocalPath,
	}
}

func (h *RemotePathMappingHandler) list(w http.ResponseWriter, r *http.Request) {
	all, err := h.store.List(r.Context())
	if err != nil {
		h.log.Error("remotepathmapping list", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]remotePathMappingResource, 0, len(all))
	for _, m := range all {
		out = append(out, toRemotePathMappingResource(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *RemotePathMappingHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	m, err := h.store.GetByID(r.Context(), id)
	if errors.Is(err, remotepathmapping.ErrNotFound) {
		writeError(w, http.StatusNotFound, "Not Found")
		return
	}
	if err != nil {
		h.log.Error("remotepathmapping get", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toRemotePathMappingResource(m))
}

func (h *RemotePathMappingHandler) create(w http.ResponseWriter, r *http.Request) {
	var body remotePathMappingResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if body.Host == "" || body.RemotePath == "" || body.LocalPath == "" {
		writeError(w, http.StatusBadRequest, "host, remotePath, and localPath are required")
		return
	}
	m, err := h.store.Create(r.Context(), remotepathmapping.Mapping{
		Host: body.Host, RemotePath: body.RemotePath, LocalPath: body.LocalPath,
	})
	if err != nil {
		h.log.Error("remotepathmapping create", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toRemotePathMappingResource(m))
}

func (h *RemotePathMappingHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	var body remotePathMappingResource
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	body.ID = id
	m := remotepathmapping.Mapping{ID: id, Host: body.Host, RemotePath: body.RemotePath, LocalPath: body.LocalPath}
	if err := h.store.Update(r.Context(), m); err != nil {
		h.log.Error("remotepathmapping update", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusAccepted, toRemotePathMappingResource(m))
}

func (h *RemotePathMappingHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.log.Error("remotepathmapping delete", slog.String("err", err.Error()))
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusOK)
}
