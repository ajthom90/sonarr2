// SPDX-License-Identifier: GPL-3.0-or-later
// Import-list provider stubs. Each stub carries its Settings struct with
// Sonarr-compatible JSON field names so /api/v3/importlist/schema exposes
// every identifier a migrating user recognizes. Fetch() and Test() return
// a clear "not yet implemented" error.
//
// Ported behaviorally from Sonarr
// (src/NzbDrone.Core/ImportLists/{AniList,Trakt,Plex,Rss,Simkl,Sonarr,MyAnimeList,Custom}/).

package importlist

import (
	"context"
	"errors"
)

// ErrStub is returned by stub provider Fetch/Test methods.
var ErrStub = errors.New("import list provider not yet implemented in sonarr2")

// --- Generic stub base -----------------------------------------------------

type stubBase struct{}

func (stubBase) Fetch(context.Context) ([]Item, error) { return nil, ErrStub }
func (stubBase) Test(context.Context) error            { return ErrStub }

// --- AniList ---------------------------------------------------------------

type AniListSettings struct {
	AuthToken string `json:"authToken" form:"text" label:"Auth Token" privacy:"apiKey"`
	ListType  string `json:"listType" form:"text" label:"List Type" placeholder:"current|completed|dropped|paused|planning|rewatching"`
	UserID    int    `json:"userId" form:"number" label:"User ID"`
}

type AniList struct {
	stubBase
	Cfg AniListSettings
}

func NewAniList() *AniList                { return &AniList{} }
func (a *AniList) Implementation() string { return "AniListImport" }
func (a *AniList) DefaultName() string    { return "AniList" }
func (a *AniList) Settings() any          { return &a.Cfg }

// --- MyAnimeList -----------------------------------------------------------

type MALSettings struct {
	AuthToken string `json:"authToken" form:"text" label:"Auth Token" privacy:"apiKey"`
	ListType  string `json:"listType" form:"text" label:"List Type" placeholder:"watching|plan_to_watch|completed|on_hold|dropped"`
	Username  string `json:"username" form:"text" label:"Username"`
}

type MyAnimeList struct {
	stubBase
	Cfg MALSettings
}

func NewMyAnimeList() *MyAnimeList            { return &MyAnimeList{} }
func (m *MyAnimeList) Implementation() string { return "MyAnimeListImport" }
func (m *MyAnimeList) DefaultName() string    { return "MyAnimeList" }
func (m *MyAnimeList) Settings() any          { return &m.Cfg }

// --- Plex Watchlist --------------------------------------------------------

type PlexWatchlistSettings struct {
	AuthToken string `json:"authToken" form:"text" label:"Plex Auth Token" required:"true" privacy:"apiKey"`
}

type PlexWatchlist struct {
	stubBase
	Cfg PlexWatchlistSettings
}

func NewPlexWatchlist() *PlexWatchlist          { return &PlexWatchlist{} }
func (p *PlexWatchlist) Implementation() string { return "PlexImport" }
func (p *PlexWatchlist) DefaultName() string    { return "Plex Watchlist" }
func (p *PlexWatchlist) Settings() any          { return &p.Cfg }

// --- Plex RSS --------------------------------------------------------------

type PlexRSSSettings struct {
	URL string `json:"url" form:"text" label:"RSS URL" required:"true"`
}

type PlexRSS struct {
	stubBase
	Cfg PlexRSSSettings
}

func NewPlexRSS() *PlexRSS                { return &PlexRSS{} }
func (p *PlexRSS) Implementation() string { return "PlexRssImport" }
func (p *PlexRSS) DefaultName() string    { return "Plex RSS" }
func (p *PlexRSS) Settings() any          { return &p.Cfg }

// --- Generic RSS -----------------------------------------------------------

type RSSSettings struct {
	URL string `json:"url" form:"text" label:"URL" required:"true"`
}

type RSS struct {
	stubBase
	Cfg RSSSettings
}

func NewRSS() *RSS                    { return &RSS{} }
func (r *RSS) Implementation() string { return "Rss" }
func (r *RSS) DefaultName() string    { return "RSS" }
func (r *RSS) Settings() any          { return &r.Cfg }

// --- Simkl -----------------------------------------------------------------

type SimklSettings struct {
	AuthToken string `json:"authToken" form:"text" label:"Auth Token" required:"true" privacy:"apiKey"`
	ListType  string `json:"listType" form:"text" label:"List Type"`
}

type Simkl struct {
	stubBase
	Cfg SimklSettings
}

func NewSimkl() *Simkl                  { return &Simkl{} }
func (s *Simkl) Implementation() string { return "SimklImport" }
func (s *Simkl) DefaultName() string    { return "Simkl" }
func (s *Simkl) Settings() any          { return &s.Cfg }

// --- Sonarr (another instance) ---------------------------------------------

type SonarrSettings struct {
	BaseURL            string   `json:"baseUrl" form:"text" label:"Sonarr URL" required:"true"`
	APIKey             string   `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	QualityProfileIds  []int    `json:"qualityProfileIds" form:"number" label:"Quality Profiles"`
	TagIds             []int    `json:"tagIds" form:"number" label:"Tags"`
	RootFolderPaths    []string `json:"rootFolderPaths" form:"text" label:"Root Folder Paths"`
	LanguageProfileIds []int    `json:"languageProfileIds" form:"number" label:"Language Profiles"`
}

type SonarrImport struct {
	stubBase
	Cfg SonarrSettings
}

func NewSonarrImport() *SonarrImport           { return &SonarrImport{} }
func (s *SonarrImport) Implementation() string { return "SonarrImport" }
func (s *SonarrImport) DefaultName() string    { return "Sonarr" }
func (s *SonarrImport) Settings() any          { return &s.Cfg }

// --- Trakt (User / List / Popular) -----------------------------------------
// Sonarr registers three distinct identifiers backed by shared OAuth state.

type TraktBaseSettings struct {
	AccessToken  string `json:"accessToken" form:"text" label:"Access Token" privacy:"apiKey"`
	RefreshToken string `json:"refreshToken" form:"text" label:"Refresh Token" privacy:"apiKey"`
	Expires      string `json:"expires" form:"text" label:"Expires"`
	AuthUser     string `json:"authUser" form:"text" label:"Auth User"`
}

type TraktUserSettings struct {
	TraktBaseSettings
	Username      string `json:"username" form:"text" label:"Username" required:"true"`
	TraktListType string `json:"traktListType" form:"text" label:"List Type" placeholder:"collection|watched|watchlist|recommendations|shows"`
	Limit         int    `json:"limit" form:"number" label:"Limit" placeholder:"100"`
}

type TraktUser struct {
	stubBase
	Cfg TraktUserSettings
}

func NewTraktUser() *TraktUser              { return &TraktUser{} }
func (t *TraktUser) Implementation() string { return "TraktUserImport" }
func (t *TraktUser) DefaultName() string    { return "Trakt User" }
func (t *TraktUser) Settings() any          { return &t.Cfg }

type TraktListSettings struct {
	TraktBaseSettings
	ListName string `json:"listName" form:"text" label:"List Name" required:"true"`
	Username string `json:"username" form:"text" label:"Username" required:"true"`
}

type TraktList struct {
	stubBase
	Cfg TraktListSettings
}

func NewTraktList() *TraktList              { return &TraktList{} }
func (t *TraktList) Implementation() string { return "TraktListImport" }
func (t *TraktList) DefaultName() string    { return "Trakt List" }
func (t *TraktList) Settings() any          { return &t.Cfg }

type TraktPopularSettings struct {
	TraktBaseSettings
	TraktListType string `json:"traktListType" form:"text" label:"List Type" placeholder:"popular|trending|anticipated|watched"`
	Limit         int    `json:"limit" form:"number" label:"Limit" placeholder:"100"`
	Genres        string `json:"genres" form:"text" label:"Genres (CSV)"`
	Years         string `json:"years" form:"text" label:"Years" placeholder:"2020-2024"`
	Ratings       string `json:"ratings" form:"text" label:"Ratings"`
	Certification string `json:"certification" form:"text" label:"Certification"`
}

type TraktPopular struct {
	stubBase
	Cfg TraktPopularSettings
}

func NewTraktPopular() *TraktPopular           { return &TraktPopular{} }
func (t *TraktPopular) Implementation() string { return "TraktPopularImport" }
func (t *TraktPopular) DefaultName() string    { return "Trakt Popular" }
func (t *TraktPopular) Settings() any          { return &t.Cfg }

// --- Custom List -----------------------------------------------------------

type CustomSettings struct {
	URL string `json:"url" form:"text" label:"URL" required:"true"`
}

type Custom struct {
	stubBase
	Cfg CustomSettings
}

func NewCustom() *Custom                 { return &Custom{} }
func (c *Custom) Implementation() string { return "CustomImport" }
func (c *Custom) DefaultName() string    { return "Custom" }
func (c *Custom) Settings() any          { return &c.Cfg }
