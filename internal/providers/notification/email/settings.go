package email

// Settings holds the configuration for an Email (SMTP) notification provider.
type Settings struct {
	Server   string `json:"server"   form:"text"     label:"Server"   required:"true" default:"smtp.example.com"`
	Port     int    `json:"port"     form:"number"   label:"Port"     required:"true" default:"587"`
	Username string `json:"username" form:"text"     label:"Username"`
	Password string `json:"password" form:"password" label:"Password"`
	From     string `json:"from"     form:"text"     label:"From"     required:"true"`
	To       string `json:"to"       form:"text"     label:"To"       required:"true"`
	UseSsl   bool   `json:"useSsl"   form:"checkbox" label:"Use SSL"`
}
