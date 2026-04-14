// SPDX-License-Identifier: GPL-3.0-or-later
// /api/v3/metadata endpoints — metadata consumer schema + list.
// Matching Sonarr's MetadataResource (src/Sonarr.Api.V3/Metadata/).

package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/metadata"
	"github.com/ajthom90/sonarr2/internal/providers"
)

type metadataSchemaResource struct {
	Implementation string          `json:"implementation"`
	Name           string          `json:"name"`
	Fields         json.RawMessage `json:"fields"`
}

// MetadataHandler exposes the metadata consumer registry over v3.
type MetadataHandler struct {
	registry *metadata.Registry
	log      *slog.Logger
}

// NewMetadataHandler constructs a MetadataHandler.
func NewMetadataHandler(reg *metadata.Registry, log *slog.Logger) *MetadataHandler {
	return &MetadataHandler{registry: reg, log: log}
}

// MountMetadata registers /api/v3/metadata routes. The list endpoint is
// read-only for now (no metadata-instance store wired yet); /schema exposes
// the registered consumers so the settings UI can build its picker.
func MountMetadata(r chi.Router, h *MetadataHandler) {
	r.Route("/api/v3/metadata", func(r chi.Router) {
		r.Get("/schema", h.schema)
		r.Get("/", h.list)
	})
}

func (h *MetadataHandler) list(w http.ResponseWriter, _ *http.Request) {
	// No instance store yet — return empty array to keep clients happy.
	writeJSON(w, http.StatusOK, []any{})
}

func (h *MetadataHandler) schema(w http.ResponseWriter, _ *http.Request) {
	names := h.registry.Names()
	out := make([]metadataSchemaResource, 0, len(names))
	for _, name := range names {
		c, err := h.registry.Build(name)
		if err != nil {
			continue
		}
		schema := providers.SchemaFor(c.Settings())
		fieldsJSON, _ := json.Marshal(schema.Fields)
		out = append(out, metadataSchemaResource{
			Implementation: name,
			Name:           c.DefaultName(),
			Fields:         json.RawMessage(fieldsJSON),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
