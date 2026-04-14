// Package flood is a stub for the Flood web UI DL client (wraps rTorrent etc.).
package flood

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

type Settings struct {
	Host       string   `json:"host" form:"text" label:"Host" required:"true"`
	Port       int      `json:"port" form:"number" label:"Port" placeholder:"3000"`
	UseSSL     bool     `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase    string   `json:"urlBase" form:"text" label:"URL Base"`
	Username   string   `json:"username" form:"text" label:"Username" required:"true"`
	Password   string   `json:"password" form:"password" label:"Password" required:"true" privacy:"password"`
	Tags       []string `json:"tags" form:"tagList" label:"Tags"`
	Directory  string   `json:"destination" form:"text" label:"Destination"`
	StartOnAdd bool     `json:"startOnAdd" form:"checkbox" label:"Start On Add"`
}

type Flood struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Flood { return &Flood{settings: s, client: client} }
func (f *Flood) Implementation() string          { return "Flood" }
func (f *Flood) DefaultName() string             { return "Flood" }
func (f *Flood) Settings() any                   { return &f.settings }
