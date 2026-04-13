# M20 — Backup

## Overview

Add automated and manual database backups. Backups are tar.gz archives containing the SQLite database file and a JSON manifest with version/schema info. Stored in `{config_dir}/Backups/` with configurable retention. API endpoints allow listing, creating, downloading, and deleting backups.

## Architecture

### Backup Package

New package `internal/backup/` with:

```go
// Manifest describes the contents of a backup archive.
type Manifest struct {
    Version       string    `json:"version"`
    DatabaseType  string    `json:"databaseType"`
    Timestamp     time.Time `json:"timestamp"`
    SchemaVersion int       `json:"schemaVersion"`
}

// Info describes a backup file on disk.
type Info struct {
    Name      string    `json:"name"`
    Path      string    `json:"-"`
    Size      int64     `json:"size"`
    Timestamp time.Time `json:"time"`
}

// Service manages backup creation, listing, and deletion.
type Service struct {
    backupDir     string
    dbPath        string      // empty for Postgres
    dbDialect     string
    schemaVersion int
    retention     int         // max backups to keep
    log           *slog.Logger
}

func New(opts Options) *Service
func (s *Service) Create(ctx context.Context) (Info, error)
func (s *Service) List() ([]Info, error)
func (s *Service) Delete(name string) error
func (s *Service) FilePath(name string) string  // for download handler
```

### Backup Contents

The tar.gz archive contains:
- `manifest.json` — version, database type, timestamp, schema version
- `sonarr2.db` — SQLite database file (only for SQLite dialect)

For Postgres users, the backup contains only the manifest (they should use `pg_dump` for database backup).

### Storage

Backups are stored in `{config_dir}/Backups/` (created automatically). Filenames follow the pattern `sonarr2_backup_YYYYMMDDHHMMSS.tar.gz`.

### Retention

After creating a backup, the service deletes the oldest backups exceeding the retention count (default 7).

### API Endpoints

**V3:**
- `GET /api/v3/system/backup` — list all backups
- `POST /api/v3/system/backup` — create a new backup, return Info
- `GET /api/v3/system/backup/{id}/download` — download backup file (id is the filename without extension)
- `DELETE /api/v3/system/backup/{id}` — delete a backup

**V6:** Same endpoints under `/api/v6/system/backup`.

### Scheduled Task

Register `Backup` as a scheduled task with a 7-day interval. The handler calls `service.Create()`.

### Configuration

| Variable | Default | Description |
|---|---|---|
| `SONARR2_BACKUP_RETENTION` | `7` | Number of backups to keep |
| `SONARR2_BACKUP_INTERVAL` | `168h` | Backup schedule interval (7 days) |

The backup directory is derived from `cfg.Paths.Config + "/Backups"`.

### Database Path Discovery

For SQLite, the DB path is extracted from the DSN (e.g., `file:./data/sonarr2.db?...` → `./data/sonarr2.db`). The backup copies this file.

## Testing

- **Create**: create a backup with a temp dir, verify tar.gz contains manifest.json and sonarr2.db
- **List**: create 2 backups, verify both listed in order
- **Delete**: create then delete, verify file removed
- **Retention**: create N+1 backups with retention=N, verify oldest is removed
- **API**: test list/create/delete endpoints

## Out of Scope

- Restore (deferred — unsafe while running, CLI-only future feature)
- Frontend backup page (API-only for now)
- Postgres database backup (users should use pg_dump)
- Encrypted backups
