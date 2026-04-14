// Package delayprofile implements Sonarr's Delay Profile feature.
//
// A delay profile introduces a waiting window before grabbing a release,
// giving time for higher-quality releases to appear. Profiles are evaluated
// in sort order; the first profile whose Tags intersect a series's tags
// applies (with a catch-all at max sort_order having no tags).
//
// PreferredProtocol chooses Usenet or Torrent when both are available;
// BypassIfHighestQuality and BypassIfAboveCustomFormatScore short-circuit
// the delay for already-optimal releases.
//
// Ported from Sonarr (src/NzbDrone.Core/Profiles/Delay/).
package delayprofile

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a delay profile does not exist.
var ErrNotFound = errors.New("delayprofile: not found")

// Protocol matches Sonarr's DownloadProtocol enum, serialized as strings.
type Protocol string

const (
	ProtocolUsenet  Protocol = "usenet"
	ProtocolTorrent Protocol = "torrent"
)

// Profile represents one delay profile row.
type Profile struct {
	ID                              int      `json:"id"`
	EnableUsenet                    bool     `json:"enableUsenet"`
	EnableTorrent                   bool     `json:"enableTorrent"`
	PreferredProtocol               Protocol `json:"preferredProtocol"`
	UsenetDelay                     int      `json:"usenetDelay"`                    // minutes
	TorrentDelay                    int      `json:"torrentDelay"`                   // minutes
	Order                           int      `json:"order"`                          // sort key (Sonarr calls it Order)
	BypassIfHighestQuality          bool     `json:"bypassIfHighestQuality"`
	BypassIfAboveCustomFormatScore  bool     `json:"bypassIfAboveCustomFormatScore"`
	MinimumCustomFormatScore        int      `json:"minimumCustomFormatScore"`
	Tags                            []int    `json:"tags"`
}

// Store provides CRUD access to delay profiles.
type Store interface {
	Create(ctx context.Context, p Profile) (Profile, error)
	GetByID(ctx context.Context, id int) (Profile, error)
	List(ctx context.Context) ([]Profile, error)
	Update(ctx context.Context, p Profile) error
	Delete(ctx context.Context, id int) error
}

// ProtocolDelay returns the configured delay (minutes) for the given protocol.
func (p Profile) ProtocolDelay(proto Protocol) int {
	if proto == ProtocolTorrent {
		return p.TorrentDelay
	}
	return p.UsenetDelay
}

// ApplicableProfile returns the first profile in profiles (assumed sorted by
// Order ascending) whose Tags intersect seriesTags. The caller should place
// the default no-tags profile last with Order=MaxInt so it catches series
// with no matching specific profile.
func ApplicableProfile(profiles []Profile, seriesTags []int) (Profile, bool) {
	tagSet := make(map[int]struct{}, len(seriesTags))
	for _, t := range seriesTags {
		tagSet[t] = struct{}{}
	}
	for _, p := range profiles {
		// Empty Tags list = catch-all (Sonarr default).
		if len(p.Tags) == 0 {
			return p, true
		}
		for _, tag := range p.Tags {
			if _, ok := tagSet[tag]; ok {
				return p, true
			}
		}
	}
	return Profile{}, false
}
