package broadcasthenet

// Settings holds the configuration for the BroadcastheNet indexer.
type Settings struct {
	ApiKey string `json:"apiKey" form:"password" label:"API Key" required:"true"  helpText:"Your BroadcastheNet API key"`
}
