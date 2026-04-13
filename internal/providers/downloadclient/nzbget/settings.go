package nzbget

// Settings holds the configuration for an NZBGet download client.
type Settings struct {
	Host     string `json:"host"     form:"text"     label:"Host"     required:"true" default:"localhost"`
	Port     int    `json:"port"     form:"number"   label:"Port"     required:"true" default:"6789"`
	Username string `json:"username" form:"text"     label:"Username" default:"nzbget"`
	Password string `json:"password" form:"password" label:"Password"`
	Category string `json:"category" form:"text"     label:"Category" default:"tv"`
	UseSsl   bool   `json:"useSsl"   form:"checkbox" label:"Use SSL"`
}
