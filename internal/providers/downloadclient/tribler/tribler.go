// Package tribler is a stub for the Tribler torrent client DL provider.
package tribler

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host     string `json:"host" form:"text" label:"Host" required:"true"`
	Port     int    `json:"port" form:"number" label:"Port" placeholder:"20100"`
	APIKey   string `json:"apiKey" form:"text" label:"API Key" privacy:"apiKey"`
	UseSSL   bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
}

type Tribler struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Tribler { return &Tribler{settings: s, client: client} }
func (t *Tribler) Implementation() string          { return "Tribler" }
func (t *Tribler) DefaultName() string             { return "Tribler" }
func (t *Tribler) Settings() any                   { return &t.settings }
