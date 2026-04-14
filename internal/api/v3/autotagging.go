// SPDX-License-Identifier: GPL-3.0-or-later
// /api/v3/autotagging endpoints — list + schema (specifications).

package v3

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// AutoTaggingHandler exposes the auto-tag rule list (stub until store lands).
type AutoTaggingHandler struct {
	log *slog.Logger
}

// NewAutoTaggingHandler constructs an AutoTaggingHandler.
func NewAutoTaggingHandler(log *slog.Logger) *AutoTaggingHandler {
	return &AutoTaggingHandler{log: log}
}

// MountAutoTagging registers /api/v3/autotagging routes.
func MountAutoTagging(r chi.Router, h *AutoTaggingHandler) {
	r.Route("/api/v3/autotagging", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/schema", h.schema)
	})
}

func (h *AutoTaggingHandler) list(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

// schema exposes the specification types available for auto-tag rules.
// Matches Sonarr's AutoTagSpecificationSchemaResource byte-for-byte so the
// UI's rule-builder picker works.
func (h *AutoTaggingHandler) schema(w http.ResponseWriter, _ *http.Request) {
	specs := []map[string]any{
		{
			"implementation":     "GenreSpecification",
			"implementationName": "Genre",
			"name":               "Genre",
			"fields": []map[string]any{
				{"name": "value", "label": "Genre", "type": "textbox"},
			},
		},
		{
			"implementation":     "SeriesStatusSpecification",
			"implementationName": "Status",
			"name":               "Status",
			"fields": []map[string]any{
				{"name": "value", "label": "Status", "type": "select",
					"selectOptions": []map[string]any{
						{"value": "continuing", "name": "Continuing"},
						{"value": "ended", "name": "Ended"},
						{"value": "upcoming", "name": "Upcoming"},
					}},
			},
		},
		{
			"implementation":     "SeriesTypeSpecification",
			"implementationName": "Series Type",
			"name":               "Series Type",
			"fields": []map[string]any{
				{"name": "value", "label": "Type", "type": "select",
					"selectOptions": []map[string]any{
						{"value": "standard", "name": "Standard"},
						{"value": "daily", "name": "Daily"},
						{"value": "anime", "name": "Anime"},
					}},
			},
		},
		{
			"implementation":     "NetworkSpecification",
			"implementationName": "Network",
			"name":               "Network",
			"fields": []map[string]any{
				{"name": "value", "label": "Network (regex)", "type": "textbox"},
			},
		},
		{
			"implementation":     "OriginalLanguageSpecification",
			"implementationName": "Original Language",
			"name":               "Original Language",
			"fields": []map[string]any{
				{"name": "value", "label": "Language", "type": "textbox"},
			},
		},
		{
			"implementation":     "YearSpecification",
			"implementationName": "Year",
			"name":               "Year",
			"fields": []map[string]any{
				{"name": "value", "label": "Year or Range (e.g. 2020-2024)", "type": "textbox"},
			},
		},
		{
			"implementation":     "RootFolderSpecification",
			"implementationName": "Root Folder",
			"name":               "Root Folder",
			"fields": []map[string]any{
				{"name": "value", "label": "Path", "type": "textbox"},
			},
		},
	}
	out, _ := json.Marshal(specs)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(out)
}
