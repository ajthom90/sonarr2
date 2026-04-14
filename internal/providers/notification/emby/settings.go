package emby

// Settings for Emby/MediaBrowser notifications.
type Settings struct {
	Host          string `json:"host" form:"text" label:"Host" required:"true"`
	Port          int    `json:"port" form:"number" label:"Port" placeholder:"8096"`
	APIKey        string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	UseSSL        bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	Notify        bool   `json:"notify" form:"checkbox" label:"Send Notifications"`
	UpdateLibrary bool   `json:"updateLibrary" form:"checkbox" label:"Update Library"`
}
