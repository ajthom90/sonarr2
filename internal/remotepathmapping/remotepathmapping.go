// Package remotepathmapping provides CRUD + path-translation for Sonarr-compatible
// Remote Path Mappings. A mapping rewrites a path reported by a remote download
// client host into a path accessible on the Sonarr host.
//
// Example: if SABnzbd reports a completed download at /downloads/foo and sonarr2
// sees that path at /mnt/nas/downloads/foo, a mapping with
// Host="sabnzbd.local", RemotePath="/downloads/", LocalPath="/mnt/nas/downloads/"
// rewrites the path on import.
//
// Ported from Sonarr (src/NzbDrone.Core/RemotePathMappings/).
package remotepathmapping

import (
	"context"
	"errors"
	"strings"
)

// ErrNotFound is returned when a mapping does not exist.
var ErrNotFound = errors.New("remotepathmapping: not found")

// Mapping is one host -> remote/local path rewrite rule.
type Mapping struct {
	ID         int    `json:"id"`
	Host       string `json:"host"`
	RemotePath string `json:"remotePath"`
	LocalPath  string `json:"localPath"`
}

// Store provides CRUD access to remote path mappings.
type Store interface {
	Create(ctx context.Context, m Mapping) (Mapping, error)
	GetByID(ctx context.Context, id int) (Mapping, error)
	List(ctx context.Context) ([]Mapping, error)
	Update(ctx context.Context, m Mapping) error
	Delete(ctx context.Context, id int) error
}

// Translate rewrites remotePath using the first mapping whose Host matches and
// whose RemotePath is a prefix of remotePath. Returns remotePath unchanged when
// no mapping matches. Host matching is case-insensitive. Trailing separators on
// the mapping's RemotePath/LocalPath are tolerated (normalized away).
func Translate(mappings []Mapping, host, remotePath string) string {
	if host == "" || remotePath == "" {
		return remotePath
	}
	hostNorm := strings.ToLower(host)
	for _, m := range mappings {
		if strings.ToLower(m.Host) != hostNorm {
			continue
		}
		rp := stripTrailingSep(m.RemotePath)
		lp := stripTrailingSep(m.LocalPath)
		// Exact match of the full path (no separator required).
		if strings.EqualFold(m.RemotePath, remotePath) ||
			strings.EqualFold(rp, remotePath) {
			return m.LocalPath
		}
		// Prefix match with separator boundary — e.g. "/downloads" matches
		// "/downloads/foo" but not "/downloadsX/foo".
		if strings.HasPrefix(remotePath, rp) {
			rest := remotePath[len(rp):]
			if rest == "" || isSep(rest[0]) {
				return lp + rest
			}
		}
	}
	return remotePath
}

// stripTrailingSep removes a single trailing "/" or "\" from a path.
func stripTrailingSep(p string) string {
	if p == "" {
		return p
	}
	if strings.HasSuffix(p, "/") || strings.HasSuffix(p, "\\") {
		return p[:len(p)-1]
	}
	return p
}

func isSep(b byte) bool { return b == '/' || b == '\\' }
