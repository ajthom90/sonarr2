// Package dstation is the Synology DownloadStation DL client. DS has a
// unified API for both Usenet (NZB) and Torrent — Sonarr registers two
// separate schema entries ("DownloadStation" torrent and "UsenetDownloadStation"
// usenet). Stubbed for now.
package dstation

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host         string `json:"host" form:"text" label:"Host" required:"true"`
	Port         int    `json:"port" form:"number" label:"Port" placeholder:"5000"`
	Username     string `json:"username" form:"text" label:"Username" required:"true"`
	Password     string `json:"password" form:"password" label:"Password" required:"true" privacy:"password"`
	UseSSL       bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase      string `json:"urlBase" form:"text" label:"URL Base"`
	Category     string `json:"category" form:"text" label:"Category"`
	Directory    string `json:"directory" form:"text" label:"Directory"`
}

// Torrent variant of Synology Download Station.
type Torrent struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func NewTorrent(s Settings, client *http.Client) *Torrent { return &Torrent{settings: s, client: client} }
func (t *Torrent) Implementation() string                 { return "DownloadStation" }
func (t *Torrent) DefaultName() string                    { return "Synology Download Station (Torrent)" }
func (t *Torrent) Settings() any                          { return &t.settings }

// Usenet variant of Synology Download Station.
type Usenet struct {
	downloadclient.StubUsenet
	settings Settings
	client   *http.Client
}

func NewUsenet(s Settings, client *http.Client) *Usenet { return &Usenet{settings: s, client: client} }
func (u *Usenet) Implementation() string                { return "UsenetDownloadStation" }
func (u *Usenet) DefaultName() string                   { return "Synology Download Station (Usenet)" }
func (u *Usenet) Settings() any                         { return &u.settings }
