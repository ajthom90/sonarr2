package blackhole

// Settings holds the configuration for a Blackhole download client.
type Settings struct {
	WatchFolder string `json:"watchFolder" form:"text" label:"Watch Folder" required:"true" helpText:"Folder where NZB/torrent files are dropped for processing"`
}
