// Package filelist is a stub indexer for FileList.io (Romanian private tracker).
package filelist

import (
	"context"
	"errors"
	"net/http"

	"github.com/ajthom90/sonarr2/internal/providers/indexer"
)

// Settings for the FileList indexer.
type Settings struct {
	BaseURL       string `json:"baseUrl" form:"text" label:"API URL" placeholder:"https://filelist.io/api.php"`
	Username      string `json:"username" form:"text" label:"Username" required:"true"`
	Passkey       string `json:"passkey" form:"text" label:"Passkey" required:"true" privacy:"apiKey"`
	Categories    []int  `json:"categories" form:"number" label:"Categories" placeholder:"21,23,27"`
	AnimeCategories []int `json:"animeCategories" form:"number" label:"Anime Categories"`
	EnableRss     bool   `json:"enableRss" form:"checkbox" label:"Enable RSS"`
	EnableAutomaticSearch bool `json:"enableAutomaticSearch" form:"checkbox" label:"Enable Auto Search"`
	EnableInteractiveSearch bool `json:"enableInteractiveSearch" form:"checkbox" label:"Enable Interactive"`
}

type FileList struct {
	settings Settings
	client   *http.Client
}

func New(s Settings, client *http.Client) *FileList { return &FileList{settings: s, client: client} }
func (f *FileList) Implementation() string          { return "FileList" }
func (f *FileList) DefaultName() string             { return "FileList" }
func (f *FileList) Settings() any                   { return &f.settings }
func (f *FileList) Protocol() indexer.DownloadProtocol { return indexer.ProtocolTorrent }
func (f *FileList) SupportsRss() bool               { return true }
func (f *FileList) SupportsSearch() bool            { return false } // stub

func (f *FileList) FetchRss(context.Context) ([]indexer.Release, error) {
	return nil, errors.New("filelist: not yet implemented")
}
func (f *FileList) Search(context.Context, indexer.SearchRequest) ([]indexer.Release, error) {
	return nil, errors.New("filelist: not yet implemented")
}
func (f *FileList) Test(context.Context) error { return nil }
