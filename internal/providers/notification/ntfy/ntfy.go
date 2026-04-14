// Package ntfy implements a notification.Notification that publishes to
// ntfy.sh (self-hosted or public instance) via HTTP POST.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Notifications/Ntfy/).
package ntfy

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ajthom90/sonarr2/internal/providers/notification"
)

// Ntfy sends notifications to one or more ntfy.sh topics.
type Ntfy struct {
	settings Settings
	client   *http.Client
}

// New constructs an Ntfy provider.
func New(settings Settings, client *http.Client) *Ntfy {
	return &Ntfy{settings: settings, client: client}
}

func (n *Ntfy) Implementation() string { return "Ntfy" }
func (n *Ntfy) DefaultName() string    { return "ntfy.sh" }
func (n *Ntfy) Settings() any          { return &n.settings }

func (n *Ntfy) Test(ctx context.Context) error {
	return n.send(ctx, "Test", "sonarr2 notification test")
}

func (n *Ntfy) OnGrab(ctx context.Context, msg notification.GrabMessage) error {
	return n.send(ctx, "Sonarr - Release Grabbed",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (n *Ntfy) OnDownload(ctx context.Context, msg notification.DownloadMessage) error {
	return n.send(ctx, "Sonarr - Download Complete",
		fmt.Sprintf("%s - %s", msg.SeriesTitle, msg.EpisodeTitle))
}

func (n *Ntfy) OnHealthIssue(ctx context.Context, msg notification.HealthMessage) error {
	return n.send(ctx, "Sonarr - Health Issue",
		fmt.Sprintf("[%s] %s", msg.Type, msg.Message))
}

func (n *Ntfy) send(ctx context.Context, title, body string) error {
	server := n.settings.ServerURL
	if server == "" {
		server = "https://ntfy.sh"
	}
	server = strings.TrimRight(server, "/")
	if len(n.settings.Topics) == 0 {
		return fmt.Errorf("ntfy: at least one topic is required")
	}

	headers := map[string]string{
		"Title": title,
	}
	if n.settings.Priority > 0 {
		headers["Priority"] = strconv.Itoa(n.settings.Priority)
	}
	if len(n.settings.Tags) > 0 {
		headers["Tags"] = strings.Join(n.settings.Tags, ",")
	}
	if n.settings.ClickURL != "" {
		headers["Click"] = n.settings.ClickURL
	}
	switch {
	case n.settings.AccessToken != "":
		headers["Authorization"] = "Bearer " + n.settings.AccessToken
	case n.settings.Username != "" && n.settings.Password != "":
		creds := base64.StdEncoding.EncodeToString([]byte(n.settings.Username + ":" + n.settings.Password))
		headers["Authorization"] = "Basic " + creds
	}

	for _, topic := range n.settings.Topics {
		url := server + "/" + strings.TrimLeft(topic, "/")
		if err := notification.Put(ctx, n.client, url, "text/plain", body, headers); err != nil {
			return fmt.Errorf("ntfy: topic %q: %w", topic, err)
		}
	}
	return nil
}
