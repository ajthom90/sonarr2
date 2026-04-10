// Package buildinfo exposes build-time metadata injected via -ldflags.
package buildinfo

// These values are overridden at link time with:
//
//	go build -ldflags="-X github.com/ajthom90/sonarr2/internal/buildinfo.Version=1.2.3 ..."
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// Info is a snapshot of build metadata for serialization.
type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

// Get returns the current build metadata snapshot.
func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
