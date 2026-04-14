// SPDX-License-Identifier: GPL-3.0-or-later
// /api/v3/manualimport — scan a folder, propose episode matches, and
// execute the import.
//
// Matches Sonarr's ManualImportController wire shape. Backing scan /
// match / move-or-hardlink implementation is pending; stubbed to keep
// clients from 404ing.

package v3

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ManualImportHandler handles /api/v3/manualimport.
type ManualImportHandler struct {
	log *slog.Logger
}

// NewManualImportHandler constructs a ManualImportHandler.
func NewManualImportHandler(log *slog.Logger) *ManualImportHandler {
	return &ManualImportHandler{log: log}
}

// MountManualImport wires /api/v3/manualimport routes.
// - GET  /?folder=&downloadId=  → proposed imports
// - POST /                     → execute imports
func MountManualImport(r chi.Router, h *ManualImportHandler) {
	r.Route("/api/v3/manualimport", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.execute)
	})
}

func (h *ManualImportHandler) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *ManualImportHandler) execute(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}
