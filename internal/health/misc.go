// SPDX-License-Identifier: GPL-3.0-or-later
// Additional health checks to match Sonarr's coverage. Each check is a
// small struct with Name() and Check(ctx). Checks return an empty slice
// when everything is healthy, or one or more Result entries with a
// severity Level when something needs attention.
//
// Ported behaviorally from Sonarr (src/NzbDrone.Core/HealthCheck/Checks/).

package health

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// --- ApiKeyValidationCheck -------------------------------------------------

// ApiKeyValidationCheck verifies the API key is non-empty and reasonably long.
// A missing or weak API key makes the v3/v6 API unreachable or trivially
// guessable. Matches Sonarr's ApiKeyValidationCheck.
type ApiKeyValidationCheck struct {
	KeyFn func() string
}

func NewApiKeyValidationCheck(keyFn func() string) *ApiKeyValidationCheck {
	return &ApiKeyValidationCheck{KeyFn: keyFn}
}

func (c *ApiKeyValidationCheck) Name() string { return "ApiKeyValidationCheck" }

func (c *ApiKeyValidationCheck) Check(_ context.Context) []Result {
	key := c.KeyFn()
	if key == "" {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: "API key is not configured — external clients cannot authenticate.",
		}}
	}
	if len(key) < 20 {
		return []Result{{
			Source: c.Name(), Type: LevelWarning,
			Message: "API key is shorter than the recommended 32 hex characters.",
		}}
	}
	return nil
}

// --- AppDataLocationCheck --------------------------------------------------

// AppDataLocationCheck warns when the configured AppData directory is not
// writable. Sonarr stores host_config, logs, and backups here; a read-only
// mount silently breaks most features.
type AppDataLocationCheck struct {
	DirFn func() string
}

func NewAppDataLocationCheck(dirFn func() string) *AppDataLocationCheck {
	return &AppDataLocationCheck{DirFn: dirFn}
}

func (c *AppDataLocationCheck) Name() string { return "AppDataLocationCheck" }

func (c *AppDataLocationCheck) Check(_ context.Context) []Result {
	dir := c.DirFn()
	if dir == "" {
		return nil
	}
	probe := filepath.Join(dir, ".sonarr2-healthprobe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: fmt.Sprintf("AppData directory %q is not writable: %v", dir, err),
		}}
	}
	_ = os.Remove(probe)
	return nil
}

// --- MountCheck ------------------------------------------------------------

// MountCheck verifies a list of paths each exist and are directories.
// Sonarr surfaces this when a user's NAS mount drops offline.
type MountCheck struct {
	PathsFn func() []string
}

func NewMountCheck(pathsFn func() []string) *MountCheck { return &MountCheck{PathsFn: pathsFn} }

func (c *MountCheck) Name() string { return "MountCheck" }

func (c *MountCheck) Check(_ context.Context) []Result {
	var out []Result
	for _, p := range c.PathsFn() {
		info, err := os.Stat(p)
		if err != nil {
			out = append(out, Result{
				Source: c.Name(), Type: LevelError,
				Message: fmt.Sprintf("Mount %q is not accessible: %v", p, err),
			})
			continue
		}
		if !info.IsDir() {
			out = append(out, Result{
				Source: c.Name(), Type: LevelError,
				Message: fmt.Sprintf("Mount %q is not a directory", p),
			})
		}
	}
	return out
}

// --- SystemTimeCheck -------------------------------------------------------

// SystemTimeCheck warns when the local clock is too far from a reference
// time. Without an NTP reference to query we rely on the HTTP Date header
// of last known reachability — for now, we just emit an informational
// Result if the system time is clearly implausible (year < 2000). A
// proper drift check would compare against an external NTP source.
type SystemTimeCheck struct{}

func NewSystemTimeCheck() *SystemTimeCheck { return &SystemTimeCheck{} }

func (c *SystemTimeCheck) Name() string { return "SystemTimeCheck" }

func (c *SystemTimeCheck) Check(_ context.Context) []Result {
	now := time.Now().UTC()
	if now.Year() < 2000 {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: fmt.Sprintf("System clock appears unset (now=%s).", now.Format(time.RFC3339)),
		}}
	}
	return nil
}

// --- RecyclingBinCheck -----------------------------------------------------

// RecyclingBinCheck verifies the configured recycle bin exists and is
// writable when the feature is enabled. An empty RecycleBinFn means the
// feature is disabled and the check is a no-op.
type RecyclingBinCheck struct {
	RecycleBinFn func() string
}

func NewRecyclingBinCheck(fn func() string) *RecyclingBinCheck {
	return &RecyclingBinCheck{RecycleBinFn: fn}
}

func (c *RecyclingBinCheck) Name() string { return "RecyclingBinCheck" }

func (c *RecyclingBinCheck) Check(_ context.Context) []Result {
	path := c.RecycleBinFn()
	if path == "" {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: fmt.Sprintf("Recycle Bin path %q is not accessible: %v", path, err),
		}}
	}
	if !info.IsDir() {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: fmt.Sprintf("Recycle Bin path %q is not a directory", path),
		}}
	}
	probe := filepath.Join(path, ".sonarr2-healthprobe")
	if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: fmt.Sprintf("Recycle Bin path %q is not writable: %v", path, err),
		}}
	}
	_ = os.Remove(probe)
	return nil
}

// --- RemotePathMappingCheck ------------------------------------------------

// RemotePathMappingCheck verifies each configured mapping's LocalPath exists.
// A broken local-path mapping silently fails import; this surfaces it.
type RemotePathMappingCheck struct {
	MappingsFn func() []struct{ Host, RemotePath, LocalPath string }
}

func NewRemotePathMappingCheck(
	fn func() []struct{ Host, RemotePath, LocalPath string },
) *RemotePathMappingCheck {
	return &RemotePathMappingCheck{MappingsFn: fn}
}

func (c *RemotePathMappingCheck) Name() string { return "RemotePathMappingCheck" }

func (c *RemotePathMappingCheck) Check(_ context.Context) []Result {
	var out []Result
	for _, m := range c.MappingsFn() {
		if m.LocalPath == "" {
			continue
		}
		info, err := os.Stat(m.LocalPath)
		if err != nil {
			out = append(out, Result{
				Source: c.Name(), Type: LevelWarning,
				Message: fmt.Sprintf("Remote path mapping %s:%s → %s: local path not accessible: %v",
					m.Host, m.RemotePath, m.LocalPath, err),
			})
			continue
		}
		if !info.IsDir() {
			out = append(out, Result{
				Source: c.Name(), Type: LevelWarning,
				Message: fmt.Sprintf("Remote path mapping %s:%s → %s: local path is not a directory",
					m.Host, m.RemotePath, m.LocalPath),
			})
		}
	}
	return out
}

// --- ProxyCheck ------------------------------------------------------------

// ProxyCheck warns when a proxy is configured but not reachable. sonarr2's
// config layer doesn't expose a proxy yet; the check no-ops until the
// General Settings → Proxy sub-section lands.
type ProxyCheck struct {
	EnabledFn func() bool
	HostFn    func() string
	PortFn    func() int
}

func NewProxyCheck(
	enabled func() bool, host func() string, port func() int,
) *ProxyCheck {
	return &ProxyCheck{EnabledFn: enabled, HostFn: host, PortFn: port}
}

func (c *ProxyCheck) Name() string { return "ProxyCheck" }

func (c *ProxyCheck) Check(_ context.Context) []Result {
	if c.EnabledFn == nil || !c.EnabledFn() {
		return nil
	}
	host := c.HostFn()
	port := c.PortFn()
	if host == "" || port == 0 {
		return []Result{{
			Source: c.Name(), Type: LevelError,
			Message: "Proxy is enabled but Host or Port is not configured.",
		}}
	}
	return nil
}

// --- RemovedSeriesCheck ----------------------------------------------------

// RemovedSeriesCheck notices series that have been marked deleted on TheTVDB
// so users can decide whether to remove them locally. Requires the metadata
// source to report status; for now it's a hook that returns nil until the
// TVDB client surfaces a per-series "deleted" flag.
type RemovedSeriesCheck struct {
	RemovedTitlesFn func() []string
}

func NewRemovedSeriesCheck(fn func() []string) *RemovedSeriesCheck {
	return &RemovedSeriesCheck{RemovedTitlesFn: fn}
}

func (c *RemovedSeriesCheck) Name() string { return "RemovedSeriesCheck" }

func (c *RemovedSeriesCheck) Check(_ context.Context) []Result {
	if c.RemovedTitlesFn == nil {
		return nil
	}
	titles := c.RemovedTitlesFn()
	if len(titles) == 0 {
		return nil
	}
	return []Result{{
		Source: c.Name(), Type: LevelWarning,
		Message: fmt.Sprintf("%d series have been removed from TheTVDB: %v", len(titles), titles),
	}}
}

// --- ImportListStatusCheck -------------------------------------------------

// ImportListStatusCheck flags when all configured import lists are failing.
// Once the import-list subsystem lands, the callback can report per-list
// status; for now it's a hook that always returns OK.
type ImportListStatusCheck struct {
	FailingFn func() []string
}

func NewImportListStatusCheck(fn func() []string) *ImportListStatusCheck {
	return &ImportListStatusCheck{FailingFn: fn}
}

func (c *ImportListStatusCheck) Name() string { return "ImportListStatusCheck" }

func (c *ImportListStatusCheck) Check(_ context.Context) []Result {
	if c.FailingFn == nil {
		return nil
	}
	failing := c.FailingFn()
	if len(failing) == 0 {
		return nil
	}
	return []Result{{
		Source: c.Name(), Type: LevelWarning,
		Message: fmt.Sprintf("Import lists reporting errors: %v", failing),
	}}
}

// --- NotificationStatusCheck -----------------------------------------------

// NotificationStatusCheck flags when any configured notification provider
// is in a failed state (e.g. repeatedly returning 4xx/5xx on dispatch).
type NotificationStatusCheck struct {
	FailingFn func() []string
}

func NewNotificationStatusCheck(fn func() []string) *NotificationStatusCheck {
	return &NotificationStatusCheck{FailingFn: fn}
}

func (c *NotificationStatusCheck) Name() string { return "NotificationStatusCheck" }

func (c *NotificationStatusCheck) Check(_ context.Context) []Result {
	if c.FailingFn == nil {
		return nil
	}
	failing := c.FailingFn()
	if len(failing) == 0 {
		return nil
	}
	return []Result{{
		Source: c.Name(), Type: LevelWarning,
		Message: fmt.Sprintf("Notification providers failing: %v", failing),
	}}
}
