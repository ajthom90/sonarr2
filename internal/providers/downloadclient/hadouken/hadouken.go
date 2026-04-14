// Package hadouken is a stub for the Hadouken torrent DL client.
package hadouken

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

type Settings struct {
	Host     string `json:"host" form:"text" label:"Host" required:"true"`
	Port     int    `json:"port" form:"number" label:"Port" placeholder:"7070"`
	UseSSL   bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase  string `json:"urlBase" form:"text" label:"URL Base" placeholder:"/api"`
	Username string `json:"username" form:"text" label:"Username"`
	Password string `json:"password" form:"password" label:"Password" privacy:"password"`
	Category string `json:"category" form:"text" label:"Category"`
}

type Hadouken struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Hadouken { return &Hadouken{settings: s, client: client} }
func (h *Hadouken) Implementation() string          { return "Hadouken" }
func (h *Hadouken) DefaultName() string             { return "Hadouken" }
func (h *Hadouken) Settings() any                   { return &h.settings }
