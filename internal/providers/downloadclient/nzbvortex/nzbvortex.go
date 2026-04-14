// Package nzbvortex is a stub for the NZBVortex Usenet download client.
// Schema identifier matches Sonarr's ("NzbVortex"). Test/Add/Remove return
// an ErrStub error until the HTTPS API client is filled in.
package nzbvortex

import (
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
)

type Settings struct {
	Host     string `json:"host" form:"text" label:"Host" required:"true"`
	Port     int    `json:"port" form:"number" label:"Port" placeholder:"4321"`
	APIKey   string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	UseSSL   bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	URLBase  string `json:"urlBase" form:"text" label:"URL Base"`
	Category string `json:"category" form:"text" label:"Category" placeholder:"sonarr"`
	Priority int    `json:"priority" form:"number" label:"Priority"`
}

type NzbVortex struct {
	downloadclient.StubUsenet
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *NzbVortex {
	return &NzbVortex{settings: s, client: client}
}

func (n *NzbVortex) Implementation() string { return "NzbVortex" }
func (n *NzbVortex) DefaultName() string    { return "NZBVortex" }
func (n *NzbVortex) Settings() any          { return &n.settings }
