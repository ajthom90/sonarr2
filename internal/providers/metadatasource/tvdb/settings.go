// Package tvdb implements MetadataSource against the TVDB v4 API.
package tvdb

// Settings holds the configuration for the TVDB v4 API client.
type Settings struct {
	ApiKey  string `json:"apiKey"  form:"password" label:"TVDB API Key" required:"true"`
	BaseURL string `json:"baseUrl" form:"text"     label:"Base URL"     default:"https://api4.thetvdb.com"`
}
