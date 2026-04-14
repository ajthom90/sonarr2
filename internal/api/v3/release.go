// SPDX-License-Identifier: GPL-3.0-or-later
// /api/v3/release endpoints — Interactive Search (list candidate releases for
// an episode/season/series) + Push (grab a selected release).
//
// Matches Sonarr's ReleaseController wire shape. Backing implementation that
// hits every configured indexer is pending; this endpoint currently returns
// an empty list so clients don't 404.

package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ReleaseHandler handles /api/v3/release.
type ReleaseHandler struct {
	log *slog.Logger
}

// NewReleaseHandler constructs a ReleaseHandler.
func NewReleaseHandler(log *slog.Logger) *ReleaseHandler { return &ReleaseHandler{log: log} }

// MountRelease registers Interactive Search routes.
// - GET  /api/v3/release?episodeId=&seasonNumber=&seriesId=  → list candidates
// - POST /api/v3/release                                     → grab release (guid+indexerId)
// - POST /api/v3/release/push                                → push a release (for external tools)
func MountRelease(r chi.Router, h *ReleaseHandler) {
	r.Route("/api/v3/release", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.grab)
		r.Post("/push", h.push)
	})
}

func (h *ReleaseHandler) list(w http.ResponseWriter, _ *http.Request) {
	// Interactive search across all enabled indexers is not yet wired.
	// Return empty list so UIs load.
	writeJSON(w, http.StatusOK, []any{})
}

type releaseGrabRequest struct {
	GUID      string `json:"guid"`
	IndexerID int    `json:"indexerId"`
}

func (h *ReleaseHandler) grab(w http.ResponseWriter, r *http.Request) {
	var body releaseGrabRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}
	if body.GUID == "" || body.IndexerID == 0 {
		writeError(w, http.StatusBadRequest, "guid and indexerId are required")
		return
	}
	// Actual grab pipeline integration is pending.
	h.log.Info("release grab (stub)",
		slog.String("guid", body.GUID),
		slog.Int("indexerId", body.IndexerID))
	writeJSON(w, http.StatusOK, map[string]any{"guid": body.GUID, "indexerId": body.IndexerID})
}

func (h *ReleaseHandler) push(w http.ResponseWriter, _ *http.Request) {
	// POST /release/push accepts a pre-fetched release from external tools.
	// Stubbed for now.
	writeJSON(w, http.StatusOK, map[string]any{})
}
