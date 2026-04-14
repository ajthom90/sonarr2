// SPDX-License-Identifier: GPL-3.0-or-later
// /api/v3/importlist endpoints — schema + list (stub) + exclusions.

package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/importlist"
	"github.com/ajthom90/sonarr2/internal/providers"
)

type importListSchemaResource struct {
	Implementation string          `json:"implementation"`
	Name           string          `json:"name"`
	Fields         json.RawMessage `json:"fields"`
}

// ImportListHandler exposes the import-list registry over v3.
type ImportListHandler struct {
	registry *importlist.Registry
	log      *slog.Logger
}

// NewImportListHandler constructs an ImportListHandler.
func NewImportListHandler(reg *importlist.Registry, log *slog.Logger) *ImportListHandler {
	return &ImportListHandler{registry: reg, log: log}
}

// MountImportList registers /api/v3/importlist routes. Read-only for now —
// the store CRUD will land as a follow-up; /schema exposes the registered
// providers so the settings UI can populate the list-add dialog.
func MountImportList(r chi.Router, h *ImportListHandler) {
	r.Route("/api/v3/importlist", func(r chi.Router) {
		r.Get("/schema", h.schema)
		r.Get("/", h.list)
	})
	r.Route("/api/v3/importlistexclusion", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			// No store wired yet — return empty array so clients don't error.
			writeJSON(w, http.StatusOK, []any{})
		})
	})
}

func (h *ImportListHandler) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (h *ImportListHandler) schema(w http.ResponseWriter, _ *http.Request) {
	names := h.registry.Names()
	out := make([]importListSchemaResource, 0, len(names))
	for _, name := range names {
		p, err := h.registry.Build(name)
		if err != nil {
			continue
		}
		schema := providers.SchemaFor(p.Settings())
		fieldsJSON, _ := json.Marshal(schema.Fields)
		out = append(out, importListSchemaResource{
			Implementation: name,
			Name:           p.DefaultName(),
			Fields:         json.RawMessage(fieldsJSON),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
