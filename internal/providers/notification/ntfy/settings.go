package ntfy

// Settings for an ntfy.sh notification provider.
type Settings struct {
	ServerURL   string   `json:"serverUrl" form:"text" label:"ntfy.sh Server URL" placeholder:"https://ntfy.sh"`
	Topics      []string `json:"topics" form:"tagList" label:"Topics"`
	Username    string   `json:"username" form:"text" label:"Username"`
	Password    string   `json:"password" form:"password" label:"Password" privacy:"password"`
	Priority    int      `json:"priority" form:"number" label:"Priority" placeholder:"3"`
	Tags        []string `json:"ntfyTags" form:"tagList" label:"Tags"`
	ClickURL    string   `json:"clickUrl" form:"text" label:"Click URL"`
	AccessToken string   `json:"accessToken" form:"text" label:"Access Token" privacy:"apiKey"`
}
