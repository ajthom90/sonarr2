// Package fanzub is a stub indexer for Fanzub (anime Usenet).
package fanzub

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

type Settings struct {
	BaseURL       string `json:"baseUrl" form:"text" label:"URL" placeholder:"https://fanzub.com/rss"`
	EnableRss     bool   `json:"enableRss" form:"checkbox" label:"Enable RSS"`
	EnableAutomaticSearch bool `json:"enableAutomaticSearch" form:"checkbox" label:"Enable Auto Search"`
	EnableInteractiveSearch bool `json:"enableInteractiveSearch" form:"checkbox" label:"Enable Interactive"`
}

type Fanzub struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Fanzub   { return &Fanzub{settings: s, client: client} }
func (f *Fanzub) Implementation() string            { return "Fanzub" }
func (f *Fanzub) DefaultName() string               { return "Fanzub" }
func (f *Fanzub) Settings() any                     { return &f.settings }
func (f *Fanzub) Protocol() indexer.DownloadProtocol { return indexer.ProtocolUsenet }
func (f *Fanzub) SupportsRss() bool                 { return true }
func (f *Fanzub) SupportsSearch() bool              { return false }
func (f *Fanzub) FetchRss(context.Context) ([]indexer.Release, error) {
	return nil, errors.New("fanzub: not yet implemented")
}
func (f *Fanzub) Search(context.Context, indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("fanzub: not yet implemented")
}
func (f *Fanzub) Test(context.Context) error { return nil }
