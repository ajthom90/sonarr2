// Package web embeds the compiled frontend files into the Go binary.
// The dist directory is populated by running `make frontend` (or `cd frontend && npm run build`).
// A placeholder index.html is committed so that `go build` always succeeds
// even when the frontend has not been built yet.
package web

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var distFS embed.FS

// DistFS returns the embedded frontend file system rooted at dist/.
// The returned fs.FS can be passed directly to http.FileServer.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
