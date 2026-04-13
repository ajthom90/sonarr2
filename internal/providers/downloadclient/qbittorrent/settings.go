package qbittorrent

// Settings holds the configuration for a qBittorrent download client.
type Settings struct {
	Host     string `json:"host"     form:"text"     label:"Host"     required:"true" default:"localhost"`
	Port     int    `json:"port"     form:"number"   label:"Port"     required:"true" default:"8080"`
	Username string `json:"username" form:"text"     label:"Username" default:"admin"`
	Password string `json:"password" form:"password" label:"Password"`
	Category string `json:"category" form:"text"     label:"Category" default:"tv"`
	UseSsl   bool   `json:"useSsl"   form:"checkbox" label:"Use SSL"`
}
