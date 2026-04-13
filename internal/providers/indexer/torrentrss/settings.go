package torrentrss

// Settings holds the configuration for a generic torrent RSS feed indexer.
type Settings struct {
	FeedURL string `json:"feedUrl" form:"text"     label:"RSS Feed URL" required:"true"  placeholder:"https://tracker.example.com/rss"`
	Cookie  string `json:"cookie"  form:"text"     label:"Cookie"       helpText:"Optional authentication cookie (e.g. uid=123; pass=abc)"`
}
