package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/importer"
	"github.com/ajthom90/sonarr2/internal/library"
)

// ScanSeriesFolderHandler scans a single series folder for new or changed media
// files and imports them into the library.
type ScanSeriesFolderHandler struct {
	library   *library.Library
	importSvc *importer.Service
	log       *slog.Logger
}

// NewScanSeriesFolderHandler creates a ScanSeriesFolderHandler wired to the
// given library and import service.
func NewScanSeriesFolderHandler(lib *library.Library, importSvc *importer.Service, log *slog.Logger) *ScanSeriesFolderHandler {
	return &ScanSeriesFolderHandler{
		library:   lib,
		importSvc: importSvc,
		log:       log,
	}
}

// Handle implements commands.Handler.
// Expected body: {"seriesId": 123}
func (h *ScanSeriesFolderHandler) Handle(ctx context.Context, cmd commands.Command) error {
	var body struct {
		SeriesID int64 `json:"seriesId"`
	}
	if err := json.Unmarshal(cmd.Body, &body); err != nil {
		return fmt.Errorf("parse body: %w", err)
	}

	series, err := h.library.Series.Get(ctx, body.SeriesID)
	if err != nil {
		return fmt.Errorf("get series: %w", err)
	}

	h.log.InfoContext(ctx, "scan_series: scanning folder",
		"seriesId", series.ID, "path", series.Path)

	return h.importSvc.ProcessFolder(ctx, series.Path, series.ID, "")
}
