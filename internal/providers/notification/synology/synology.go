// Package synology implements the Synology Indexer notification provider.
// It refreshes the Synology DSM media indexer so newly imported episodes
// are picked up by Video Station / DS Video.
//
// On a real DSM install this invokes `synoindex -A` via local shell; we
// expose a stub matching Sonarr's settings schema so users migrating from
// Sonarr see their config, with a clear "requires DSM" error at runtime.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Synology/).
package synology

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings matches Sonarr's Synology Indexer settings.
type Settings struct {
	UpdateLibrary bool `json:"updateLibrary" form:"checkbox" label:"Update Library"`
}

type Synology struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Synology { return &Synology{settings: s, client: client} }

func (s *Synology) Implementation() string { return "SynologyIndexer" }
func (s *Synology) DefaultName() string    { return "Synology Indexer" }
func (s *Synology) Settings() any          { return &s.settings }

var errNotImplemented = errors.New("synology: indexer requires DSM-local synoindex binary; not yet wired in sonarr2")

func (s *Synology) Test(context.Context) error { return errNotImplemented }
func (s *Synology) OnGrab(context.Context, notification.GrabMessage) error {
	return nil
}
func (s *Synology) OnDownload(context.Context, notification.DownloadMessage) error {
	if !s.settings.UpdateLibrary {
		return nil
	}
	return errNotImplemented
}
func (s *Synology) OnHealthIssue(context.Context, notification.HealthMessage) error {
	return nil
}
