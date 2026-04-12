package newznab

// Settings holds the configuration for a Newznab-compatible indexer.
type Settings struct {
	BaseURL    string `json:"baseUrl"    form:"text"     label:"URL"        required:"true"  placeholder:"https://indexer.example.com"`
	ApiPath    string `json:"apiPath"    form:"text"     label:"API Path"   default:"/api"`
	ApiKey     string `json:"apiKey"     form:"password" label:"API Key"    required:"true"`
	Categories string `json:"categories" form:"text"     label:"Categories" helpText:"Comma-separated category IDs (e.g., 5030,5040)"`
}
