package deluge

// Settings holds the configuration for a Deluge download client.
type Settings struct {
	Host     string `json:"host"     form:"text"     label:"Host"     required:"true" default:"localhost"`
	Port     int    `json:"port"     form:"number"   label:"Port"     required:"true" default:"8112"`
	Password string `json:"password" form:"password" label:"Password" required:"true" default:"deluge"`
	UseSsl   bool   `json:"useSsl"   form:"checkbox" label:"Use SSL"`
}
