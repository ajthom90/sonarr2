// Package freebox is a stub for the Freebox Download DL client (French ISP).
package freebox

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

type Settings struct {
	Host       string `json:"host" form:"text" label:"Host" placeholder:"mafreebox.freebox.fr"`
	Port       int    `json:"port" form:"number" label:"Port" placeholder:"443"`
	UseSSL     bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	APIBase    string `json:"apiBase" form:"text" label:"API Base"`
	APIVersion string `json:"apiVersion" form:"text" label:"API Version"`
	AppID      string `json:"appId" form:"text" label:"App ID" privacy:"apiKey"`
	AppToken   string `json:"appToken" form:"text" label:"App Token" privacy:"apiKey"`
	Directory  string `json:"destinationDirectory" form:"text" label:"Destination"`
	Category   string `json:"category" form:"text" label:"Category"`
	AddPaused  bool   `json:"addPaused" form:"checkbox" label:"Add Paused"`
}

type Download struct {
	downloadclient.StubTorrent
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Download {
	return &Download{settings: s, client: client}
}
func (f *Download) Implementation() string { return "FreeboxDownload" }
func (f *Download) DefaultName() string    { return "Freebox Download" }
func (f *Download) Settings() any          { return &f.settings }
