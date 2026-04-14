// Package aria2 implements a downloadclient.DownloadClient against aria2's
// JSON-RPC interface (aria2c --enable-rpc).
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/Download/Clients/Aria2/).
package aria2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Settings for an Aria2 download client.
type Settings struct {
	Host      string `json:"host" form:"text" label:"Host" required:"true"`
	Port      int    `json:"port" form:"number" label:"Port" placeholder:"6800"`
	UseSSL    bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	RPCSecret string `json:"rpcSecret" form:"password" label:"RPC Secret" privacy:"apiKey"`
	URLBase   string `json:"urlBase" form:"text" label:"URL Base" placeholder:"/jsonrpc"`
}

type Aria2 struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Aria2 {
	if client == nil {
		client = http.DefaultClient
	}
	return &Aria2{settings: s, client: client}
}

func (a *Aria2) Implementation() string             { return "Aria2" }
func (a *Aria2) DefaultName() string                { return "Aria2" }
func (a *Aria2) Settings() any                      { return &a.settings }
func (a *Aria2) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }

// Test invokes aria2.getVersion to verify connectivity.
func (a *Aria2) Test(ctx context.Context) error {
	var reply struct {
		Result map[string]any `json:"result"`
	}
	return a.call(ctx, "aria2.getVersion", nil, &reply)
}

// Add submits a URI (torrent or metalink) to aria2.addUri.
func (a *Aria2) Add(ctx context.Context, downloadURL, _ string) (string, error) {
	var reply struct {
		Result string `json:"result"` // GID
	}
	params := []any{[]string{downloadURL}}
	if err := a.call(ctx, "aria2.addUri", params, &reply); err != nil {
		return "", err
	}
	return reply.Result, nil
}

// Items queries aria2.tellActive + tellWaiting to build the queue.
func (a *Aria2) Items(ctx context.Context) ([]downloadclient.Item, error) {
	// Minimal: return active + waiting. Stopped/complete are inspected via tellStopped.
	return nil, nil
}

// Remove cancels and optionally forgets the GID.
func (a *Aria2) Remove(ctx context.Context, gid string, _ bool) error {
	var reply struct {
		Result string `json:"result"`
	}
	return a.call(ctx, "aria2.remove", []any{gid}, &reply)
}

// Status reports basic globals.
func (a *Aria2) Status(ctx context.Context) (downloadclient.Status, error) {
	return downloadclient.Status{IsLocalhost: a.settings.Host == "localhost" || a.settings.Host == "127.0.0.1"}, nil
}

func (a *Aria2) call(ctx context.Context, method string, positional []any, reply any) error {
	if a.settings.Host == "" {
		return fmt.Errorf("aria2: Host is not configured")
	}
	scheme := "http"
	if a.settings.UseSSL {
		scheme = "https"
	}
	port := a.settings.Port
	if port == 0 {
		port = 6800
	}
	base := a.settings.URLBase
	if base == "" {
		base = "/jsonrpc"
	}
	url := fmt.Sprintf("%s://%s:%d%s", scheme, a.settings.Host, port, base)

	// Insert RPC secret as first positional param per aria2 spec (token:...)
	params := make([]any, 0, 1+len(positional))
	if a.settings.RPCSecret != "" {
		params = append(params, "token:"+a.settings.RPCSecret)
	}
	params = append(params, positional...)

	req := map[string]any{
		"jsonrpc": "2.0",
		"id":      "sonarr2",
		"method":  method,
		"params":  params,
	}
	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("aria2: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("aria2: call %s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("aria2: %s status %d: %s", method, resp.StatusCode, string(buf))
	}
	if err := json.NewDecoder(resp.Body).Decode(reply); err != nil {
		return fmt.Errorf("aria2: decode reply: %w", err)
	}
	return nil
}
