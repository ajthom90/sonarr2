// Package pneumatic is the Kodi Pneumatic Usenet script provider (writes .nzb
// files to a folder that Kodi's pneumatic script picks up). Stub for now.
package pneumatic

import (
	"github.com/ajthom90/sonarr2/internal/providers/downloadclient"
	"net/http"
)

type Settings struct {
	NzbFolder    string `json:"nzbFolder" form:"text" label:"NZB Folder" required:"true"`
	StrmFolder   string `json:"strmFolder" form:"text" label:"STRM Folder"`
}

type Pneumatic struct {
	downloadclient.StubUsenet
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *Pneumatic { return &Pneumatic{settings: s, client: client} }
func (p *Pneumatic) Implementation() string          { return "Pneumatic" }
func (p *Pneumatic) DefaultName() string             { return "Pneumatic" }
func (p *Pneumatic) Settings() any                   { return &p.settings }
