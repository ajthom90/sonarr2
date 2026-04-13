package discord

// Settings holds the configuration for a Discord notification provider.
type Settings struct {
	WebhookURL string `json:"webhookUrl" form:"text" label:"Webhook URL" required:"true" placeholder:"https://discord.com/api/webhooks/..."`
}
