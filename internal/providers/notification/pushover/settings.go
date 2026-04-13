package pushover

// Settings holds the configuration for a Pushover notification provider.
type Settings struct {
	UserKey  string `json:"userKey"  form:"text"     label:"User Key"  required:"true"`
	ApiToken string `json:"apiToken" form:"password" label:"API Token" required:"true"`
}
