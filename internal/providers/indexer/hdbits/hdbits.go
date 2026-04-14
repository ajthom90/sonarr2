// Package hdbits is a stub indexer for HDBits (private tracker).
package hdbits

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

type Settings struct {
	BaseURL                 string `json:"baseUrl" form:"text" label:"API URL" placeholder:"https://hdbits.org"`
	Username                string `json:"username" form:"text" label:"Username" required:"true"`
	APIKey                  string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	Categories              []int  `json:"categories" form:"number" label:"Categories"`
	Codecs                  []int  `json:"codecs" form:"number" label:"Codecs"`
	Mediums                 []int  `json:"mediums" form:"number" label:"Mediums"`
	EnableRss               bool   `json:"enableRss" form:"checkbox" label:"Enable RSS"`
	EnableAutomaticSearch   bool   `json:"enableAutomaticSearch" form:"checkbox" label:"Enable Auto Search"`
	EnableInteractiveSearch bool   `json:"enableInteractiveSearch" form:"checkbox" label:"Enable Interactive"`
}

type HDBits struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *HDBits    { return &HDBits{settings: s, client: client} }
func (h *HDBits) Implementation() string             { return "HDBits" }
func (h *HDBits) DefaultName() string                { return "HDBits" }
func (h *HDBits) Settings() any                      { return &h.settings }
func (h *HDBits) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }
func (h *HDBits) SupportsRss() bool                  { return true }
func (h *HDBits) SupportsSearch() bool               { return false }
func (h *HDBits) FetchRss(context.Context) ([]indexer.Release, error) {
	return nil, errors.New("hdbits: not yet implemented")
}
func (h *HDBits) Search(context.Context, indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("hdbits: not yet implemented")
}
func (h *HDBits) Test(context.Context) error { return nil }
