package iptorrents

// Settings holds the configuration for the IPTorrents indexer.
type Settings struct {
	FeedURL string `json:"feedUrl" form:"text"     label:"RSS Feed URL" required:"true"  placeholder:"https://iptorrents.com/t.rss?..."`
	Cookie  string `json:"cookie"  form:"password" label:"Cookie"       required:"true"  helpText:"Authentication cookie (uid=...; pass=...)"`
}
