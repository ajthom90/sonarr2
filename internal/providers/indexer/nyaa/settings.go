package nyaa

// Settings holds the configuration for the Nyaa indexer.
type Settings struct {
	BaseURL    string `json:"baseUrl"    form:"text"  label:"URL"        default:"https://nyaa.si"  placeholder:"https://nyaa.si"`
	Categories string `json:"categories" form:"text"  label:"Categories" helpText:"Nyaa category IDs (e.g., 1_2 for anime)"`
	Filter     string `json:"filter"     form:"text"  label:"Filter"     helpText:"0=no filter, 1=no remakes, 2=trusted only" default:"0"`
}
