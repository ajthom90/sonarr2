package v6

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ajthom90/sonarr2/internal/buildinfo"
)

// startupTime records when the process started.
var startupTime = time.Now().UTC()

// v6PoolPinger is the minimal interface for DB status reporting.
type v6PoolPinger interface {
	Dialect() string
	Ping(ctx context.Context) error
}

// v6SystemStatusResource is the v6 JSON shape for system/status.
// Drops legacy fields: isNetCore.
type v6SystemStatusResource struct {
	AppName                string `json:"appName"`
	InstanceName           string `json:"instanceName"`
	Version                string `json:"version"`
	BuildTime              string `json:"buildTime"`
	IsDebug                bool   `json:"isDebug"`
	IsProduction           bool   `json:"isProduction"`
	IsAdmin                bool   `json:"isAdmin"`
	IsUserInteractive      bool   `json:"isUserInteractive"`
	StartupPath            string `json:"startupPath"`
	AppData                string `json:"appData"`
	OsName                 string `json:"osName"`
	OsVersion              string `json:"osVersion"`
	IsLinux                bool   `json:"isLinux"`
	IsOsx                  bool   `json:"isOsx"`
	IsWindows              bool   `json:"isWindows"`
	IsDocker               bool   `json:"isDocker"`
	Mode                   string `json:"mode"`
	Branch                 string `json:"branch"`
	Authentication         string `json:"authentication"`
	MigrationVersion       int    `json:"migrationVersion"`
	URLBase                string `json:"urlBase"`
	RuntimeVersion         string `json:"runtimeVersion"`
	RuntimeName            string `json:"runtimeName"`
	StartTime              string `json:"startTime"`
	PackageVersion         string `json:"packageVersion"`
	PackageAuthor          string `json:"packageAuthor"`
	PackageUpdateMechanism string `json:"packageUpdateMechanism"`
	DatabaseVersion        string `json:"databaseVersion"`
	DatabaseType           string `json:"databaseType"`
}

func mountSystemStatus(r chi.Router, pool v6PoolPinger) {
	r.Get("/system/status", func(w http.ResponseWriter, r *http.Request) {
		info := buildinfo.Get()

		startupPath, _ := os.Getwd()
		appData, _ := os.UserConfigDir()

		dbType := "sqLite"
		if pool != nil && pool.Dialect() == "postgres" {
			dbType = "postgresql"
		}

		goos := runtime.GOOS
		isLinux := goos == "linux"
		isOsx := goos == "darwin"
		isWindows := goos == "windows"
		osName := goos

		_, isDocker := os.Stat("/.dockerenv")

		runtimeVer := runtime.Version()
		if len(runtimeVer) > 2 && runtimeVer[:2] == "go" {
			runtimeVer = runtimeVer[2:]
		}

		resp := v6SystemStatusResource{
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
			IsLinux:                isLinux,
			IsOsx:                  isOsx,
			IsWindows:              isWindows,
			IsDocker:               isDocker == nil,
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
	})
}
