// Package twitter is a Twitter notification provider stub.
//
// Twitter's v2 API requires OAuth 1.0a with user context (consumer key +
// consumer secret + access token + access token secret) plus app approval
// for posting. We register the provider so it appears in the settings UI
// schema with the expected fields; the actual Test/OnX methods return a
// helpful "not yet implemented" error until OAuth is wired in.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Twitter/).
package twitter

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings matches Sonarr's Twitter settings schema.
type Settings struct {
	ConsumerKey       string `json:"consumerKey" form:"text" label:"Consumer Key" required:"true" privacy:"apiKey"`
	ConsumerSecret    string `json:"consumerSecret" form:"text" label:"Consumer Secret" required:"true" privacy:"apiKey"`
	AccessToken       string `json:"accessToken" form:"text" label:"Access Token" required:"true" privacy:"apiKey"`
	AccessTokenSecret string `json:"accessTokenSecret" form:"text" label:"Access Token Secret" required:"true" privacy:"apiKey"`
	Mention           string `json:"mention" form:"text" label:"Mention"`
	DirectMessage     bool   `json:"directMessage" form:"checkbox" label:"Direct Message"`
}

type Twitter struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Twitter { return &Twitter{settings: s, client: client} }

func (t *Twitter) Implementation() string { return "Twitter" }
func (t *Twitter) DefaultName() string    { return "Twitter" }
func (t *Twitter) Settings() any          { return &t.settings }

var errNotImplemented = errors.New("twitter: OAuth 1.0a flow not yet implemented in sonarr2")

func (t *Twitter) Test(context.Context) error                             { return errNotImplemented }
func (t *Twitter) OnGrab(context.Context, notification.GrabMessage) error { return errNotImplemented }
func (t *Twitter) OnDownload(context.Context, notification.DownloadMessage) error {
	return errNotImplemented
}
func (t *Twitter) OnHealthIssue(context.Context, notification.HealthMessage) error {
	return errNotImplemented
}
