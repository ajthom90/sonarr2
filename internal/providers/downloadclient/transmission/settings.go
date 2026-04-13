package transmission

// Settings holds the configuration for a Transmission download client.
type Settings struct {
	Host     string `json:"host"    form:"text"     label:"Host"     required:"true" default:"localhost"`
	Port     int    `json:"port"    form:"number"   label:"Port"     required:"true" default:"9091"`
	UrlBase  string `json:"urlBase" form:"text"     label:"URL Base" default:"/transmission/"`
	Username string `json:"username" form:"text"    label:"Username"`
	Password string `json:"password" form:"password" label:"Password"`
	UseSsl   bool   `json:"useSsl"  form:"checkbox" label:"Use SSL"`
}
