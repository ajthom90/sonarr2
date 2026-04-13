package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ajthom90/sonarr2/internal/commands"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

// RefreshMonitoredDownloadsHandler polls all enabled download clients for
// completed items and enqueues a ProcessDownload command for each one that
// has not already been queued.
type RefreshMonitoredDownloadsHandler struct {
	dcStore    downloadclient.InstanceStore
	dcRegistry *downloadclient.Registry
	cmdQueue   commands.Queue
	history    history.Store
	log        *slog.Logger
}

// NewRefreshMonitoredDownloadsHandler creates a RefreshMonitoredDownloadsHandler
// wired to the given stores, registry, queue, and logger.
func NewRefreshMonitoredDownloadsHandler(
	dcStore downloadclient.InstanceStore,
	dcRegistry *downloadclient.Registry,
	cmdQueue commands.Queue,
	histStore history.Store,
	log *slog.Logger,
) *RefreshMonitoredDownloadsHandler {
	return &RefreshMonitoredDownloadsHandler{
		dcStore:    dcStore,
		dcRegistry: dcRegistry,
		cmdQueue:   cmdQueue,
		history:    histStore,
		log:        log,
	}
}

// Handle implements commands.Handler.
// For each enabled download client it fetches the item queue, and for every
// completed item that has a matching grab history entry it enqueues a
// ProcessDownload command (deduplicated by "ProcessDownload:<downloadID>").
func (h *RefreshMonitoredDownloadsHandler) Handle(ctx context.Context, cmd commands.Command) error {
	// 1. List all download client instances.
	instances, err := h.dcStore.List(ctx)
	if err != nil {
		return fmt.Errorf("refresh_downloads: list instances: %w", err)
	}

	for _, inst := range instances {
		if !inst.Enable {
			continue
		}

		// 2. Look up the factory for this implementation.
		factory, err := h.dcRegistry.Get(inst.Implementation)
		if err != nil {
			h.log.WarnContext(ctx, "refresh_downloads: unknown implementation",
				"implementation", inst.Implementation, "err", err)
			continue
		}

		// 3. Instantiate and apply stored settings.
		dc := factory()
		if len(inst.Settings) > 0 {
			if err := json.Unmarshal(inst.Settings, dc.Settings()); err != nil {
				h.log.WarnContext(ctx, "refresh_downloads: unmarshal settings",
					"clientName", inst.Name, "err", err)
				continue
			}
		}

		// 4. Fetch the current item queue.
		items, err := dc.Items(ctx)
		if err != nil {
			h.log.WarnContext(ctx, "refresh_downloads: items error",
				"clientName", inst.Name, "err", err)
			continue
		}

		for _, item := range items {
			if item.Status != "completed" {
				continue
			}

			// 5. Find the original grab from history to resolve seriesID.
			entries, err := h.history.FindByDownloadID(ctx, item.DownloadID)
			if err != nil {
				h.log.WarnContext(ctx, "refresh_downloads: find by download id",
					"downloadID", item.DownloadID, "err", err)
				continue
			}
			if len(entries) == 0 {
				h.log.DebugContext(ctx, "refresh_downloads: no history entry for download",
					"downloadID", item.DownloadID)
				continue
			}
			seriesID := entries[0].SeriesID

			// 6. Dedup: skip if ProcessDownload already queued for this download.
			dedupKey := "ProcessDownload:" + item.DownloadID
			_, found, err := h.cmdQueue.FindDuplicate(ctx, dedupKey)
			if err != nil {
				h.log.WarnContext(ctx, "refresh_downloads: find duplicate",
					"dedupKey", dedupKey, "err", err)
				continue
			}
			if found {
				h.log.DebugContext(ctx, "refresh_downloads: ProcessDownload already queued",
					"downloadID", item.DownloadID)
				continue
			}

			// 7. Enqueue ProcessDownload.
			body, err := json.Marshal(struct {
				DownloadFolder string `json:"downloadFolder"`
				SeriesID       int64  `json:"seriesId"`
				DownloadID     string `json:"downloadId"`
			}{
				DownloadFolder: item.OutputPath,
				SeriesID:       seriesID,
				DownloadID:     item.DownloadID,
			})
			if err != nil {
				h.log.ErrorContext(ctx, "refresh_downloads: marshal body", "err", err)
				continue
			}

			if _, err := h.cmdQueue.Enqueue(ctx, "ProcessDownload", body,
				commands.PriorityNormal, commands.TriggerScheduled, dedupKey); err != nil {
				h.log.WarnContext(ctx, "refresh_downloads: enqueue ProcessDownload",
					"downloadID", item.DownloadID, "err", err)
			} else {
				h.log.InfoContext(ctx, "refresh_downloads: enqueued ProcessDownload",
					"downloadID", item.DownloadID, "seriesID", seriesID)
			}
		}
	}

	return nil
}
