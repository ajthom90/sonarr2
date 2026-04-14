// Package utorrent is a stub for the µTorrent / uTorrent Web UI DL client.
package utorrent

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host           string `json:"host" form:"text" label:"Host" required:"true"`
	Port           int    `json:"port" form:"number" label:"Port" placeholder:"8080"`
	UseSSL         bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase        string `json:"urlBase" form:"text" label:"URL Base" placeholder:"/gui"`
	Username       string `json:"username" form:"text" label:"Username"`
	Password       string `json:"password" form:"password" label:"Password" privacy:"password"`
	Category       string `json:"tvCategory" form:"text" label:"Category" placeholder:"tv-sonarr"`
	RecentPriority int    `json:"recentTvPriority" form:"number" label:"Recent Priority"`
	OlderPriority  int    `json:"olderTvPriority" form:"number" label:"Older Priority"`
	IntialState    int    `json:"intialState" form:"number" label:"Initial State"`
}

type UTorrent struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *UTorrent { return &UTorrent{settings: s, client: client} }
func (u *UTorrent) Implementation() string          { return "UTorrent" }
func (u *UTorrent) DefaultName() string             { return "uTorrent" }
func (u *UTorrent) Settings() any                   { return &u.settings }
