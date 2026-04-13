// Package nzbget implements a downloadclient.DownloadClient for NZBGet.
// NZBGet exposes a JSON-RPC API at POST /jsonrpc.
package nzbget

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

// NZBGet is a downloadclient.DownloadClient for NZBGet.
type NZBGet struct {
	settings Settings
	client   *http.Client
}

// New constructs an NZBGet download client. Pass nil for client to use http.DefaultClient.
func New(settings Settings, client *http.Client) *NZBGet {
	if client == nil {
		client = http.DefaultClient
	}
	return &NZBGet{settings: settings, client: client}
}

// Implementation satisfies providers.Provider.
func (n *NZBGet) Implementation() string { return "NZBGet" }

// DefaultName satisfies providers.Provider.
func (n *NZBGet) DefaultName() string { return "NZBGet" }

// Settings satisfies providers.Provider.
func (n *NZBGet) Settings() any { return &n.settings }

// Protocol satisfies downloadclient.DownloadClient.
func (n *NZBGet) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }

// Add submits an NZB URL to NZBGet via the "append" JSON-RPC method.
// Returns the NZB ID as a string.
func (n *NZBGet) Add(ctx context.Context, nzbURL string, title string) (string, error) {
	// params: NZBFilename, NZBContent (URL), Category, Priority, AddToTop, AddPaused, DupeKey, DupeScore, DupeMode
	params := []interface{}{title, nzbURL, n.settings.Category, 0, false, false, "", 0, "SCORE"}
	var result int64
	if err := n.call(ctx, "append", params, &result); err != nil {
		return "", err
	}
	if result <= 0 {
		return "", fmt.Errorf("nzbget: append returned ID %d", result)
	}
	return fmt.Sprintf("%d", result), nil
}

// Items returns the current download queue via the "listgroups" JSON-RPC method.
func (n *NZBGet) Items(ctx context.Context) ([]downloadclient.Item, error) {
	var result []listGroupsItem
	if err := n.call(ctx, "listgroups", []interface{}{0}, &result); err != nil {
		return nil, err
	}

	items := make([]downloadclient.Item, 0, len(result))
	for _, g := range result {
		items = append(items, downloadclient.Item{
			DownloadID: fmt.Sprintf("%d", g.NZBID),
			Title:      g.NZBFilename,
			Status:     g.Status,
			TotalSize:  g.FileSizeMB * 1024 * 1024,
			Remaining:  g.RemainingSizeMB * 1024 * 1024,
			OutputPath: g.DestDir,
		})
	}
	return items, nil
}

// Remove deletes an NZB from the queue via the "editqueue" JSON-RPC method.
func (n *NZBGet) Remove(ctx context.Context, downloadID string, deleteData bool) error {
	action := "GroupDelete"
	if deleteData {
		action = "GroupFinalDelete"
	}
	params := []interface{}{action, "", []string{downloadID}}
	var result bool
	return n.call(ctx, "editqueue", params, &result)
}

// Status returns the download client's overall state.
func (n *NZBGet) Status(_ context.Context) (downloadclient.Status, error) {
	isLocalhost := n.settings.Host == "localhost" || n.settings.Host == "127.0.0.1"
	return downloadclient.Status{IsLocalhost: isLocalhost}, nil
}

// Test verifies NZBGet is reachable by calling the "version" JSON-RPC method.
func (n *NZBGet) Test(ctx context.Context) error {
	var result string
	return n.call(ctx, "version", []interface{}{}, &result)
}

// call executes a JSON-RPC request and unmarshals the result field.
func (n *NZBGet) call(ctx context.Context, method string, params interface{}, result interface{}) error {
	reqBody := rpcRequest{Method: method, Params: params}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("nzbget: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.jsonrpcURL(), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("nzbget: failed to build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if n.settings.Username != "" {
		req.SetBasicAuth(n.settings.Username, n.settings.Password)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("nzbget: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("nzbget: jsonrpc returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("nzbget: failed to read response: %w", err)
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return fmt.Errorf("nzbget: failed to parse response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("nzbget: rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if result != nil {
		return json.Unmarshal(rpcResp.Result, result)
	}
	return nil
}

// jsonrpcURL builds the JSON-RPC endpoint URL.
func (n *NZBGet) jsonrpcURL() string {
	scheme := "http"
	if n.settings.UseSsl {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/jsonrpc", scheme, n.settings.Host, n.settings.Port)
}

// --- JSON-RPC types ---

type rpcRequest struct {
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type listGroupsItem struct {
	NZBID           int64  `json:"NZBID"`
	NZBFilename     string `json:"NZBFilename"`
	Status          string `json:"Status"`
	FileSizeMB      int64  `json:"FileSizeMB"`
	RemainingSizeMB int64  `json:"RemainingSizeMB"`
	DestDir         string `json:"DestDir"`
}
