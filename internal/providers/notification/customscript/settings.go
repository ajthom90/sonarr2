package customscript

// Settings holds the configuration for a CustomScript notification provider.
type Settings struct {
	Path      string `json:"path"      form:"text" label:"Script Path" required:"true" placeholder:"/config/scripts/notify.sh"`
	Arguments string `json:"arguments" form:"text" label:"Arguments"`
}
