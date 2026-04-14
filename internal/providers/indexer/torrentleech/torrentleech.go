// Package torrentleech is a stub indexer for Torrentleech.
package torrentleech

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

type Settings struct {
	BaseURL    string `json:"baseUrl" form:"text" label:"API URL" placeholder:"https://rss.torrentleech.org"`
	APIKey     string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	EnableRss  bool   `json:"enableRss" form:"checkbox" label:"Enable RSS"`
	EnableAutomaticSearch bool `json:"enableAutomaticSearch" form:"checkbox" label:"Enable Auto Search"`
	EnableInteractiveSearch bool `json:"enableInteractiveSearch" form:"checkbox" label:"Enable Interactive"`
}

type Torrentleech struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Torrentleech  { return &Torrentleech{settings: s, client: client} }
func (t *Torrentleech) Implementation() string           { return "Torrentleech" }
func (t *Torrentleech) DefaultName() string              { return "Torrentleech" }
func (t *Torrentleech) Settings() any                    { return &t.settings }
func (t *Torrentleech) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }
func (t *Torrentleech) SupportsRss() bool                { return true }
func (t *Torrentleech) SupportsSearch() bool             { return false }
func (t *Torrentleech) FetchRss(context.Context) ([]indexer.Release, error) {
	return nil, errors.New("torrentleech: not yet implemented")
}
func (t *Torrentleech) Search(context.Context, indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("torrentleech: not yet implemented")
}
func (t *Torrentleech) Test(context.Context) error { return nil }
