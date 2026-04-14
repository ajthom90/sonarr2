// Package rtorrent is a stub for the rTorrent (ruTorrent) XML-RPC DL client.
// Full XML-RPC client is pending.
package rtorrent

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host         string `json:"host" form:"text" label:"Host" required:"true"`
	Port         int    `json:"port" form:"number" label:"Port" placeholder:"8080"`
	UseSSL       bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase      string `json:"urlBase" form:"text" label:"URL Base" placeholder:"/RPC2"`
	Username     string `json:"username" form:"text" label:"Username"`
	Password     string `json:"password" form:"password" label:"Password" privacy:"password"`
	Category     string `json:"tvCategory" form:"text" label:"Category" placeholder:"tv-sonarr"`
	Directory    string `json:"tvDirectory" form:"text" label:"Directory"`
	RecentPriority int   `json:"recentTvPriority" form:"number" label:"Recent Priority"`
	OlderPriority  int   `json:"olderTvPriority" form:"number" label:"Older Priority"`
	AddStopped     bool  `json:"addStopped" form:"checkbox" label:"Add Paused"`
}

type RTorrent struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *RTorrent { return &RTorrent{settings: s, client: client} }
func (r *RTorrent) Implementation() string          { return "RTorrent" }
func (r *RTorrent) DefaultName() string             { return "rTorrent" }
func (r *RTorrent) Settings() any                   { return &r.settings }
