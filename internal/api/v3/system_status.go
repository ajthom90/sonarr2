package v3

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
)

// startupTime records when the process started so we can report it in the
// status response the same way Sonarr does.
var startupTime = time.Now().UTC()

// systemStatusResource is the Sonarr v3 JSON shape for /api/v3/system/status.
// Field names and casing match the real Sonarr API exactly so existing clients
// (Overseerr, Prowlarr, nzb360, etc.) work without modification.
type systemStatusResource struct {
	AppName               string `json:"appName"`
	InstanceName          string `json:"instanceName"`
	Version               string `json:"version"`
	BuildTime             string `json:"buildTime"`
	IsDebug               bool   `json:"isDebug"`
	IsProduction          bool   `json:"isProduction"`
	IsAdmin               bool   `json:"isAdmin"`
	IsUserInteractive     bool   `json:"isUserInteractive"`
	StartupPath           string `json:"startupPath"`
	AppData               string `json:"appData"`
	OsName                string `json:"osName"`
	OsVersion             string `json:"osVersion"`
	IsNetCore             bool   `json:"isNetCore"`
	IsLinux               bool   `json:"isLinux"`
	IsOsx                 bool   `json:"isOsx"`
	IsWindows             bool   `json:"isWindows"`
	IsDocker              bool   `json:"isDocker"`
	Mode                  string `json:"mode"`
	Branch                string `json:"branch"`
	Authentication        string `json:"authentication"`
	MigrationVersion      int    `json:"migrationVersion"`
	URLBase               string `json:"urlBase"`
	RuntimeVersion        string `json:"runtimeVersion"`
	RuntimeName           string `json:"runtimeName"`
	StartTime             string `json:"startTime"`
	PackageVersion        string `json:"packageVersion"`
	PackageAuthor         string `json:"packageAuthor"`
	PackageUpdateMechanism string `json:"packageUpdateMechanism"`
	DatabaseVersion       string `json:"databaseVersion"`
	DatabaseType          string `json:"databaseType"`
}

// PoolPinger is the subset of the DB pool needed for status reporting. The v3
// package uses this interface rather than importing the full db package.
type PoolPinger interface {
	Dialect() string
	Ping(ctx context.Context) error
}

// SystemStatusHandler handles /api/v3/system/status.
type SystemStatusHandler struct {
	pool PoolPinger
}

// NewSystemStatusHandler constructs a handler. pool may be nil; in that case
// database connectivity is reported as unknown.
func NewSystemStatusHandler(pool PoolPinger) *SystemStatusHandler {
	return &SystemStatusHandler{pool: pool}
}

// MountSystemStatus registers /api/v3/system/status on r.
func MountSystemStatus(r chi.Router, h *SystemStatusHandler) {
	r.Get("/api/v3/system/status", h.handle)
}

func (h *SystemStatusHandler) handle(w http.ResponseWriter, r *http.Request) {
	info := buildinfo.Get()

	startupPath, _ := os.Getwd()
	appData, _ := os.UserConfigDir()

	dbType := "sqLite"
	if h.pool != nil && h.pool.Dialect() == "postgres" {
		dbType = "postgresql"
	}

	goos := runtime.GOOS
	isLinux := goos == "linux"
	isOsx := goos == "darwin"
	isWindows := goos == "windows"
	osName := goos

	// Detect Docker: /.dockerenv exists in almost every container image.
	_, isDocker := os.Stat("/.dockerenv")

	runtimeVer := runtime.Version()
	// runtime.Version() returns e.g. "go1.23.4" — strip the leading "go".
	if len(runtimeVer) > 2 && runtimeVer[:2] == "go" {
		runtimeVer = runtimeVer[2:]
	}

	resp := systemStatusResource{
		AppName:                "sonarr2",
		InstanceName:           "sonarr2",
		Version:                info.Version,
		BuildTime:              info.Date,
		IsDebug:                false,
		IsProduction:           true,
		IsAdmin:                false,
		IsUserInteractive:      false,
		StartupPath:            startupPath,
		AppData:                appData,
		OsName:                 osName,
		OsVersion:              "",
		IsNetCore:              false,
		IsLinux:                isLinux,
		IsOsx:                  isOsx,
		IsWindows:              isWindows,
		IsDocker:               isDocker == nil, // Stat returns nil error when file exists
		Mode:                   "console",
		Branch:                 "main",
		Authentication:         "forms",
		MigrationVersion:       217,
		URLBase:                "",
		RuntimeVersion:         runtimeVer,
		RuntimeName:            "go",
		StartTime:              startupTime.Format(time.RFC3339),
		PackageVersion:         info.Version,
		PackageAuthor:          "",
		PackageUpdateMechanism: "docker",
		DatabaseVersion:        "",
		DatabaseType:           dbType,
	}

	writeJSON(w, http.StatusOK, resp)
}
