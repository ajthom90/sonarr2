package telegram

// Settings holds the configuration for a Telegram notification provider.
type Settings struct {
	BotToken string `json:"botToken" form:"password" label:"Bot Token" required:"true"`
	ChatID   string `json:"chatId"   form:"text"     label:"Chat ID"   required:"true"`
}
