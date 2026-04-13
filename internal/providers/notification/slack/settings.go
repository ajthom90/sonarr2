package slack

// Settings holds the configuration for a Slack notification provider.
type Settings struct {
	WebhookURL string `json:"webhookUrl" form:"text"     label:"Webhook URL" required:"true" placeholder:"https://hooks.slack.com/services/..."`
	Channel    string `json:"channel"    form:"text"     label:"Channel"     placeholder:"#sonarr"`
}
