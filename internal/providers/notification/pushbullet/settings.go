package pushbullet

// Settings holds the configuration for a Pushbullet notification provider.
type Settings struct {
	APIKey      string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
	DeviceIds   string `json:"deviceIds" form:"text" label:"Device IDs" placeholder:"Comma-separated device idens; empty = all devices"`
	ChannelTags string `json:"channelTags" form:"text" label:"Channel Tags" placeholder:"Comma-separated channel tags"`
	SenderID    string `json:"senderId" form:"text" label:"Sender ID"`
}
