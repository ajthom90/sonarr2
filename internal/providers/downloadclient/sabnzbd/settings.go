package sabnzbd

// Settings holds the configuration for a SABnzbd download client.
type Settings struct {
	Host     string `json:"host"     form:"text"     label:"Host"     required:"true" default:"localhost"`
	Port     int    `json:"port"     form:"number"   label:"Port"     required:"true" default:"8080"`
	ApiKey   string `json:"apiKey"   form:"password" label:"API Key"  required:"true"`
	UseSsl   bool   `json:"useSsl"   form:"checkbox" label:"Use SSL"`
	Category string `json:"category" form:"text"     label:"Category" default:"tv"`
}
