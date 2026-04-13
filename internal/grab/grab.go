// Package grab picks the right download client for a release and sends it.
// It also records a "grabbed" history entry and publishes a ReleasesGrabbed
// event so other parts of the system can react.
package grab

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ajthom90/sonarr2/internal/events"
	"github.com/ajthom90/sonarr2/internal/history"
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// ReleasesGrabbed is published after a release has been successfully sent to a
// download client. Other subsystems (e.g. import pipeline) can listen for this
// to track in-progress downloads.
type ReleasesGrabbed struct {
	SeriesID   int64
	EpisodeIDs []int64
	Title      string
	DownloadID string
}

// Service sends releases to download clients and records grab history.
type Service struct {
	dcStore    downloadclient.InstanceStore
	dcRegistry *downloadclient.Registry
	history    history.Store
	bus        events.Bus
	log        *slog.Logger
}

// New constructs a GrabService.
func New(
	dcStore downloadclient.InstanceStore,
	dcRegistry *downloadclient.Registry,
	historyStore history.Store,
	bus events.Bus,
	log *slog.Logger,
) *Service {
	return &Service{
		dcStore:    dcStore,
		dcRegistry: dcRegistry,
		history:    historyStore,
		bus:        bus,
		log:        log,
	}
}

// Grab sends release to the highest-priority enabled download client whose
// protocol matches the release, records a "grabbed" history entry for each
// episode, and publishes a ReleasesGrabbed event.
func (s *Service) Grab(
	ctx context.Context,
	release indexer.Release,
	seriesID int64,
	episodeIDs []int64,
	qualityName string,
) error {
	// 1. List all configured download clients.
	all, err := s.dcStore.List(ctx)
	if err != nil {
		return fmt.Errorf("grab: list download clients: %w", err)
	}

	// 2. Filter to enabled clients whose protocol matches the release.
	var candidates []downloadclient.Instance
	for _, inst := range all {
		if !inst.Enable {
			continue
		}
		factory, err := s.dcRegistry.Get(inst.Implementation)
		if err != nil {
			s.log.WarnContext(ctx, "no factory for download client implementation",
				"implementation", inst.Implementation, "id", inst.ID)
			continue
		}
		dc := factory()
		if dc.Protocol() == release.Protocol {
			candidates = append(candidates, inst)
		}
	}

	if len(candidates) == 0 {
		return fmt.Errorf("grab: no enabled download client found for protocol %q", release.Protocol)
	}

	// 3. Pick the highest-priority (lowest numeric value) client.
	best := candidates[0]
	for _, c := range candidates[1:] {
		if c.Priority < best.Priority {
			best = c
		}
	}

	// 4. Instantiate the chosen client and overlay its stored settings.
	factory, err := s.dcRegistry.Get(best.Implementation)
	if err != nil {
		return fmt.Errorf("grab: factory for %q: %w", best.Implementation, err)
	}
	dc := factory()
	if len(best.Settings) > 0 {
		if err := json.Unmarshal(best.Settings, dc.Settings()); err != nil {
			return fmt.Errorf("grab: unmarshal settings for %q: %w", best.Implementation, err)
		}
	}

	// 5. Send the release.
	downloadID, err := dc.Add(ctx, release.DownloadURL, release.Title)
	if err != nil {
		return fmt.Errorf("grab: add to download client %q: %w", best.Name, err)
	}

	s.log.InfoContext(ctx, "release grabbed",
		"title", release.Title,
		"client", best.Name,
		"downloadID", downloadID,
	)

	// 6. Record a history entry for each episode.
	for _, epID := range episodeIDs {
		entry := history.Entry{
			EpisodeID:   epID,
			SeriesID:    seriesID,
			SourceTitle: release.Title,
			QualityName: qualityName,
			EventType:   history.EventGrabbed,
			DownloadID:  downloadID,
			Data:        json.RawMessage(`{}`),
		}
		if _, err := s.history.Create(ctx, entry); err != nil {
			s.log.WarnContext(ctx, "failed to record grab history",
				"episodeID", epID, "error", err)
		}
	}

	// 7. Publish event.
	evt := ReleasesGrabbed{
		SeriesID:   seriesID,
		EpisodeIDs: episodeIDs,
		Title:      release.Title,
		DownloadID: downloadID,
	}
	if err := s.bus.Publish(ctx, evt); err != nil {
		s.log.WarnContext(ctx, "failed to publish ReleasesGrabbed event", "error", err)
	}

	return nil
}
