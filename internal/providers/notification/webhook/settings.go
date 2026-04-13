package webhook

// Settings holds the configuration for a generic Webhook notification provider.
type Settings struct {
	URL    string `json:"url"    form:"text" label:"URL"    required:"true" placeholder:"https://example.com/hook"`
	Method string `json:"method" form:"text" label:"Method" default:"POST"`
}
