// Package vuze is a stub for the Vuze (Azureus) Web Remote UI DL client.
package vuze

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host         string `json:"host" form:"text" label:"Host" required:"true"`
	Port         int    `json:"port" form:"number" label:"Port" placeholder:"9091"`
	UseSSL       bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase      string `json:"urlBase" form:"text" label:"URL Base"`
	Username     string `json:"username" form:"text" label:"Username"`
	Password     string `json:"password" form:"password" label:"Password" privacy:"password"`
	Category     string `json:"tvCategory" form:"text" label:"Category" placeholder:"tv-sonarr"`
	Directory    string `json:"tvDirectory" form:"text" label:"Directory"`
	RecentPriority int `json:"recentTvPriority" form:"number" label:"Recent Priority"`
	OlderPriority  int `json:"olderTvPriority" form:"number" label:"Older Priority"`
	AddPaused      bool `json:"addPaused" form:"checkbox" label:"Add Paused"`
}

type Vuze struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Vuze { return &Vuze{settings: s, client: client} }
func (v *Vuze) Implementation() string          { return "Vuze" }
func (v *Vuze) DefaultName() string             { return "Vuze" }
func (v *Vuze) Settings() any                   { return &v.settings }
