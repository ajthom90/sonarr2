// Package rqbit is a stub for the RQBit Rust torrent client DL provider.
package rqbit

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	Host      string `json:"host" form:"text" label:"Host" required:"true"`
	Port      int    `json:"port" form:"number" label:"Port" placeholder:"3030"`
	UseSSL    bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase   string `json:"urlBase" form:"text" label:"URL Base"`
	Directory string `json:"destination" form:"text" label:"Destination"`
}

type RQBit struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *RQBit { return &RQBit{settings: s, client: client} }
func (r *RQBit) Implementation() string          { return "RQBit" }
func (r *RQBit) DefaultName() string             { return "RQBit" }
func (r *RQBit) Settings() any                   { return &r.settings }
