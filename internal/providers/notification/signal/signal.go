// Package signal implements Signal messenger notifications via signal-cli's
// REST wrapper (https://github.com/bbernhard/signal-cli-rest-api).
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Signal/).
package signal

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Settings for Signal notifications.
type Settings struct {
	ServerURL  string `json:"serverUrl" form:"text" label:"signal-cli REST URL" required:"true"`
	SenderNumber string `json:"senderNumber" form:"text" label:"Sender Number" required:"true" placeholder:"+15551234567"`
	Recipients string `json:"recipients" form:"text" label:"Recipients" required:"true" placeholder:"CSV phone numbers"`
	AuthUsername string `json:"authUsername" form:"text" label:"Auth Username"`
	AuthPassword string `json:"authPassword" form:"password" label:"Auth Password" privacy:"password"`
}

type Signal struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Signal { return &Signal{settings: s, client: client} }

func (s *Signal) Implementation() string { return "Signal" }
func (s *Signal) DefaultName() string    { return "Signal" }
func (s *Signal) Settings() any          { return &s.settings }

func (s *Signal) Test(ctx context.Context) error {
	return s.send(ctx, "sonarr2 notification test")
}

func (s *Signal) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return s.send(ctx, fmt.Sprintf("Sonarr - Release Grabbed: %s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (s *Signal) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return s.send(ctx, fmt.Sprintf("Sonarr - Download Complete: %s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (s *Signal) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return s.send(ctx, fmt.Sprintf("Sonarr - Health Issue [%s]: %s", msg.Type, msg.Message))
}

func (s *Signal) send(ctx context.Context, body string) error {
	if s.settings.ServerURL == "" || s.settings.SenderNumber == "" || s.settings.Recipients == "" {
		return fmt.Errorf("signal: ServerURL, SenderNumber and Recipients are required")
	}
	server := strings.TrimRight(s.settings.ServerURL, "/")
	recips := []string{}
	for _, r := range strings.Split(s.settings.Recipients, ",") {
		if r = strings.TrimSpace(r); r != "" {
			recips = append(recips, r)
		}
	}
	payload := map[string]any{
		"message":    body,
		"number":     s.settings.SenderNumber,
		"recipients": recips,
	}
	headers := map[string]string{}
	if s.settings.AuthUsername != "" {
		headers["Authorization"] = "Basic " + encodeBase64(s.settings.AuthUsername+":"+s.settings.AuthPassword)
	}
	return notification.PostJSONWithHeaders(ctx, s.client, server+"/v2/send", payload, headers)
}

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
