// Package mailgun implements a Mailgun-based email notification provider.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Mailgun/).
package mailgun

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for Mailgun.
type Settings struct {
	APIKey        string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	UseEuEndpoint bool   `json:"useEuEndpoint" form:"checkbox" label:"Use EU Endpoint"`
	Domain        string `json:"domain" form:"text" label:"Domain" required:"true"`
	FromAddress   string `json:"from" form:"text" label:"From Address" required:"true"`
	Recipients    string `json:"recipients" form:"text" label:"Recipients" required:"true" placeholder:"CSV"`
}

type Mailgun struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Mailgun { return &Mailgun{settings: s, client: client} }

func (m *Mailgun) Implementation() string { return "Mailgun" }
func (m *Mailgun) DefaultName() string    { return "Mailgun" }
func (m *Mailgun) Settings() any          { return &m.settings }

func (m *Mailgun) Test(ctx context.Context) error {
	return m.send(ctx, "sonarr2 test", "This is a test email from sonarr2.")
}

func (m *Mailgun) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return m.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (m *Mailgun) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return m.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (m *Mailgun) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return m.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (m *Mailgun) send(ctx context.Context, subject, text string) error {
	if m.settings.APIKey == "" || m.settings.Domain == "" {
		return fmt.Errorf("mailgun: APIKey and Domain are required")
	}
	base := "https://api.mailgun.net"
	if m.settings.UseEuEndpoint {
		base = "https://api.eu.mailgun.net"
	}
	endpoint := fmt.Sprintf("%s/v3/%s/messages", base, url.PathEscape(m.settings.Domain))

	v := url.Values{}
	v.Set("from", m.settings.FromAddress)
	for _, r := range strings.Split(m.settings.Recipients, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			v.Add("to", r)
		}
	}
	v.Set("subject", subject)
	v.Set("text", text)

	headers := map[string]string{
		"Authorization": "Basic " + basicAuth("api", m.settings.APIKey),
	}
	return notification.PostForm(ctx, m.client, endpoint, v.Encode(), headers)
}

func basicAuth(user, pass string) string { return encodeBase64(user + ":" + pass) }

func encodeBase64(s string) string {
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	in := []byte(s)
	var out strings.Builder
	for i := 0; i < len(in); i += 3 {
		n := 0
		take := 0
		for j := 0; j < 3 && i+j < len(in); j++ {
			n |= int(in[i+j]) << (8 * (2 - j))
			take++
		}
		out.WriteByte(alpha[(n>>18)&0x3f])
		out.WriteByte(alpha[(n>>12)&0x3f])
		if take > 1 {
			out.WriteByte(alpha[(n>>6)&0x3f])
		} else {
			out.WriteByte('=')
		}
		if take > 2 {
			out.WriteByte(alpha[n&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}
