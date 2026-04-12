package tvdb

// tvdbLoginRequest is the body sent to POST /v4/login.
type tvdbLoginRequest struct {
	APIKey string `json:"apikey"`
}

// tvdbLoginResponse is the response from POST /v4/login.
type tvdbLoginResponse struct {
	Status string `json:"status"`
	Data   struct {
		Token string `json:"token"`
	} `json:"data"`
}

// tvdbSearchResponse is the response from GET /v4/search.
type tvdbSearchResponse struct {
	Status string             `json:"status"`
	Data   []tvdbSearchResult `json:"data"`
}

// tvdbSearchResult is a single result in the search response.
// Note: tvdb_id is a string in the search response.
type tvdbSearchResult struct {
	TvdbID   string `json:"tvdb_id"`
	Name     string `json:"name"`
	Year     string `json:"year"`
	Overview string `json:"overview"`
	Status   struct {
		Name string `json:"name"`
	} `json:"status"`
	Network string `json:"network"`
	Slug    string `json:"slug"`
}

// tvdbSeriesResponse is the response from GET /v4/series/{id}.
type tvdbSeriesResponse struct {
	Status string     `json:"status"`
	Data   tvdbSeries `json:"data"`
}

// tvdbSeries is the detailed series object.
type tvdbSeries struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Year            string     `json:"year"`
	Overview        string     `json:"overview"`
	Status          tvdbStatus `json:"status"`
	OriginalNetwork struct {
		Name string `json:"name"`
	} `json:"originalNetwork"`
	AverageRuntime int    `json:"averageRuntime"`
	AirsTime       string `json:"airsTime"`
	Slug           string `json:"slug"`
	Genres         []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

// tvdbStatus is the status sub-object used in both series and search results.
type tvdbStatus struct {
	Name string `json:"name"`
}

// tvdbEpisodesResponse is the response from GET /v4/series/{id}/episodes/default.
type tvdbEpisodesResponse struct {
	Status string `json:"status"`
	Data   struct {
		Episodes []tvdbEpisode `json:"episodes"`
	} `json:"data"`
	Links tvdbLinks `json:"links"`
}

// tvdbEpisode is a single episode record.
type tvdbEpisode struct {
	ID             int64  `json:"id"`
	SeasonNumber   int    `json:"seasonNumber"`
	Number         int    `json:"number"`
	AbsoluteNumber int    `json:"absoluteNumber"`
	Name           string `json:"name"`
	Overview       string `json:"overview"`
	Aired          string `json:"aired"`
}

// tvdbLinks is the pagination object at the bottom of list responses.
type tvdbLinks struct {
	Prev       *string `json:"prev"`
	Self       string  `json:"self"`
	Next       *string `json:"next"`
	TotalItems int     `json:"total_items"`
	PageSize   int     `json:"page_size"`
}
