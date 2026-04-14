package plex

// Settings for a Plex Media Server notification provider.
type Settings struct {
	Host          string `json:"host" form:"text" label:"Host" required:"true"`
	Port          int    `json:"port" form:"number" label:"Port" placeholder:"32400"`
	AuthToken     string `json:"authToken" form:"text" label:"Auth Token" required:"true" privacy:"apiKey"`
	UseSSL        bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	UpdateLibrary bool   `json:"updateLibrary" form:"checkbox" label:"Update Library"`
	MapFrom       string `json:"mapFrom" form:"text" label:"Map From"`
	MapTo         string `json:"mapTo" form:"text" label:"Map To"`
}
