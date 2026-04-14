package notifiarr

// Settings for Notifiarr notifications.
type Settings struct {
	APIKey string `json:"apiKey" form:"text" label:"API Key" required:"true" privacy:"apiKey"`
}
