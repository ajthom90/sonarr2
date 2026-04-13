// Package backup provides tar.gz backup creation and management for the
// sonarr2 database and configuration.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Manifest describes the contents of a backup archive.
type Manifest struct {
	Version   string    `json:"version"`
	DBDialect string    `json:"databaseType"`
	Timestamp time.Time `json:"timestamp"`
}

// Info describes a backup file on disk.
type Info struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Timestamp time.Time `json:"time"`
}

// Options configures the backup Service.
type Options struct {
	BackupDir  string // directory for backup files
	DBPath     string // SQLite file path, empty for Postgres
	DBDialect  string // "sqlite" or "postgres"
	AppVersion string // application version for manifest
	Retention  int    // max backups to keep (default 7)
	Log        *slog.Logger
}

// Service manages backup creation, listing, and deletion.
type Service struct {
	opts Options
}

// New creates a Service.
func New(opts Options) *Service {
	if opts.Retention <= 0 {
		opts.Retention = 7
	}
	return &Service{opts: opts}
}

// Create generates a new tar.gz backup archive and returns its Info.
// It also enforces the retention policy, removing the oldest backups
// if the total count exceeds opts.Retention.
func (s *Service) Create(_ context.Context) (Info, error) {
	if err := os.MkdirAll(s.opts.BackupDir, 0o755); err != nil {
		return Info{}, fmt.Errorf("backup: create dir: %w", err)
	}

	filename := fmt.Sprintf("sonarr2_backup_%s.tar.gz", time.Now().Format("20060102150405"))
	fullPath := filepath.Join(s.opts.BackupDir, filename)

	f, err := os.Create(fullPath)
	if err != nil {
		return Info{}, fmt.Errorf("backup: create file: %w", err)
	}

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	// Write manifest.json
	manifest := Manifest{
		Version:   s.opts.AppVersion,
		DBDialect: s.opts.DBDialect,
		Timestamp: time.Now().UTC(),
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		tw.Close()
		gw.Close()
		f.Close()
		os.Remove(fullPath)
		return Info{}, fmt.Errorf("backup: marshal manifest: %w", err)
	}
	if err := addBytesToTar(tw, "manifest.json", manifestData); err != nil {
		tw.Close()
		gw.Close()
		f.Close()
		os.Remove(fullPath)
		return Info{}, fmt.Errorf("backup: write manifest: %w", err)
	}

	// Include SQLite DB if configured
	if s.opts.DBDialect == "sqlite" && s.opts.DBPath != "" {
		if err := addFileToTar(tw, s.opts.DBPath, "sonarr2.db"); err != nil {
			tw.Close()
			gw.Close()
			f.Close()
			os.Remove(fullPath)
			return Info{}, fmt.Errorf("backup: write db: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		gw.Close()
		f.Close()
		os.Remove(fullPath)
		return Info{}, fmt.Errorf("backup: close tar: %w", err)
	}
	if err := gw.Close(); err != nil {
		f.Close()
		os.Remove(fullPath)
		return Info{}, fmt.Errorf("backup: close gzip: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(fullPath)
		return Info{}, fmt.Errorf("backup: close file: %w", err)
	}

	stat, err := os.Stat(fullPath)
	if err != nil {
		return Info{}, fmt.Errorf("backup: stat: %w", err)
	}

	info := Info{
		Name:      filename,
		Size:      stat.Size(),
		Timestamp: stat.ModTime(),
	}

	if err := s.enforceRetention(); err != nil {
		if s.opts.Log != nil {
			s.opts.Log.Warn("backup: retention enforcement failed", "err", err)
		}
	}

	return info, nil
}

// List returns all backup archives, sorted newest first.
func (s *Service) List(_ context.Context) ([]Info, error) {
	entries, err := os.ReadDir(s.opts.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Info{}, nil
		}
		return nil, fmt.Errorf("backup: read dir: %w", err)
	}

	var infos []Info
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "sonarr2_backup_") || !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		fi, err := entry.Info()
		if err != nil {
			continue
		}
		infos = append(infos, Info{
			Name:      name,
			Size:      fi.Size(),
			Timestamp: fi.ModTime(),
		})
	}

	// Sort newest first
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Timestamp.After(infos[j].Timestamp)
	})

	return infos, nil
}

// Delete removes the named backup archive.
func (s *Service) Delete(_ context.Context, name string) error {
	fullPath, err := s.FilePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("backup: remove: %w", err)
	}
	return nil
}

// FilePath returns the absolute path to a backup file, validating that
// the name resolves inside BackupDir (preventing path traversal).
func (s *Service) FilePath(name string) (string, error) {
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return "", fmt.Errorf("backup: invalid name %q", name)
	}
	abs, err := filepath.Abs(filepath.Join(s.opts.BackupDir, name))
	if err != nil {
		return "", fmt.Errorf("backup: resolve path: %w", err)
	}
	dir, err := filepath.Abs(s.opts.BackupDir)
	if err != nil {
		return "", fmt.Errorf("backup: resolve dir: %w", err)
	}
	if !strings.HasPrefix(abs, dir+string(filepath.Separator)) {
		return "", fmt.Errorf("backup: path traversal detected for name %q", name)
	}
	return abs, nil
}

// enforceRetention deletes the oldest backups beyond the retention limit.
func (s *Service) enforceRetention() error {
	infos, err := s.List(context.Background())
	if err != nil {
		return err
	}
	if len(infos) <= s.opts.Retention {
		return nil
	}
	// infos is newest-first; extras are at the end
	extras := infos[s.opts.Retention:]
	for _, extra := range extras {
		if err := s.Delete(context.Background(), extra.Name); err != nil {
			return err
		}
	}
	return nil
}

// addFileToTar writes an on-disk file into the tar archive under archiveName.
func addFileToTar(tw *tar.Writer, fsPath, archiveName string) error {
	f, err := os.Open(fsPath)
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}
	hdr := &tar.Header{
		Name:    archiveName,
		Size:    stat.Size(),
		Mode:    0o644,
		ModTime: stat.ModTime(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}

// addBytesToTar writes a byte slice into the tar archive under name.
func addBytesToTar(tw *tar.Writer, name string, data []byte) error {
	hdr := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0o644,
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}
