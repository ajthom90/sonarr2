package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/importer"
)

// ProcessDownloadHandler triggers the import pipeline when a download completes.
// It unwraps the command body and delegates to the importer.Service.
type ProcessDownloadHandler struct {
	importSvc *importer.Service
}

// NewProcessDownloadHandler creates a ProcessDownloadHandler wired to the given
// import service.
func NewProcessDownloadHandler(importSvc *importer.Service) *ProcessDownloadHandler {
	return &ProcessDownloadHandler{importSvc: importSvc}
}

// Handle implements commands.Handler.
// Expected body: {"downloadFolder": "/path", "seriesId": 123, "downloadId": "abc"}
func (h *ProcessDownloadHandler) Handle(ctx context.Context, cmd commands.Command) error {
	var body struct {
		DownloadFolder string `json:"downloadFolder"`
		SeriesID       int64  `json:"seriesId"`
		DownloadID     string `json:"downloadId"`
	}
	if err := json.Unmarshal(cmd.Body, &body); err != nil {
		return fmt.Errorf("parse body: %w", err)
	}
	return h.importSvc.ProcessFolder(ctx, body.DownloadFolder, body.SeriesID, body.DownloadID)
}
