package kodi

// Settings for Kodi (XBMC) notifications via JSON-RPC.
type Settings struct {
	Host          string `json:"host" form:"text" label:"Host" required:"true"`
	Port          int    `json:"port" form:"number" label:"Port" placeholder:"8080"`
	Username      string `json:"username" form:"text" label:"Username"`
	Password      string `json:"password" form:"password" label:"Password" privacy:"password"`
	UseSSL        bool   `json:"useSsl" form:"checkbox" label:"Use SSL"`
	DisplayTime   int    `json:"displayTime" form:"number" label:"Display Time" placeholder:"5"`
	Notify        bool   `json:"notify" form:"checkbox" label:"GUI Notification"`
	UpdateLibrary bool   `json:"updateLibrary" form:"checkbox" label:"Update Library"`
	CleanLibrary  bool   `json:"cleanLibrary" form:"checkbox" label:"Clean Library"`
	AlwaysUpdate  bool   `json:"alwaysUpdate" form:"checkbox" label:"Always Update (even if playing)"`
}
