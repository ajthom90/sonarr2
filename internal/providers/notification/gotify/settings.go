package gotify

// Settings holds the configuration for a Gotify notification provider.
type Settings struct {
	ServerURL string `json:"serverUrl" form:"text"     label:"Server URL" required:"true" placeholder:"https://gotify.example.com"`
	AppToken  string `json:"appToken"  form:"password" label:"App Token"  required:"true"`
}
