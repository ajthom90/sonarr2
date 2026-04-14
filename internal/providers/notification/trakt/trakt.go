// Package trakt is a Trakt notification / scrobbling provider.
//
// Trakt uses OAuth 2.0 with device auth + refresh tokens. The provider
// registers the settings schema so users can see it in the UI, but the
// Test/OnX methods surface a clear "not yet implemented" error until the
// device-auth flow is wired in.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Trakt/).
package trakt

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings matches Sonarr's Trakt notification settings.
type Settings struct {
	AccessToken  string `json:"accessToken" form:"text" label:"Access Token" privacy:"apiKey"`
	RefreshToken string `json:"refreshToken" form:"text" label:"Refresh Token" privacy:"apiKey"`
	Expires      string `json:"expires" form:"text" label:"Token Expiry"`
	AuthUser     string `json:"authUser" form:"text" label:"Auth User"`
}

type Trakt struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Trakt { return &Trakt{settings: s, client: client} }

func (t *Trakt) Implementation() string { return "Trakt" }
func (t *Trakt) DefaultName() string    { return "Trakt" }
func (t *Trakt) Settings() any          { return &t.settings }

var errNotImplemented = errors.New("trakt: OAuth device-auth flow not yet implemented in sonarr2")

func (t *Trakt) Test(context.Context) error                             { return errNotImplemented }
func (t *Trakt) OnGrab(context.Context, notification.GrabMessage) error { return errNotImplemented }
func (t *Trakt) OnDownload(context.Context, notification.DownloadMessage) error {
	return errNotImplemented
}
func (t *Trakt) OnHealthIssue(context.Context, notification.HealthMessage) error {
	return errNotImplemented
}
