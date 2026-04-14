// Package recyclebin implements Sonarr's Recycle Bin behavior.
//
// When the Recycle Bin path is configured in host_config, file deletions
// move the file/folder into the recycle bin instead of permanently removing
// it. A scheduled Cleanup task then purges entries older than
// RecycleBinCleanupDays.
//
// Empty recycle bin path = permanent delete. CleanupDays=0 = never purge.
//
// Ported from Sonarr (src/NzbDrone.Core/MediaFiles/RecycleBinProvider.cs).
package recyclebin

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// DeleteFile moves path into recycleBin when non-empty, or permanently
// deletes path otherwise. Parent folders in the recycle bin are created
// as needed. Returns the final path (inside the recycle bin, or empty when
// deleted permanently).
func DeleteFile(path, recycleBin, subfolder string) (string, error) {
	if recycleBin == "" {
		if err := os.Remove(path); err != nil {
			if os.IsNotExist(err) {
				return "", nil
			}
			return "", fmt.Errorf("recyclebin: permanent delete: %w", err)
		}
		return "", nil
	}

	dest := filepath.Join(recycleBin, subfolder, filepath.Base(path))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("recyclebin: mkdir: %w", err)
	}
	if err := os.Rename(path, dest); err != nil {
		// Rename may fail across filesystems — fall back to copy+remove.
		if err := copyFile(path, dest); err != nil {
			return "", fmt.Errorf("recyclebin: cross-fs copy: %w", err)
		}
		if err := os.Remove(path); err != nil {
			return "", fmt.Errorf("recyclebin: remove source after copy: %w", err)
		}
	}
	// Touch the file so cleanup date reflects recycle time, not original mtime.
	now := time.Now()
	_ = os.Chtimes(dest, now, now)
	return dest, nil
}

// DeleteFolder moves folder path into recycleBin when non-empty, or
// permanently deletes it otherwise.
func DeleteFolder(path, recycleBin string) (string, error) {
	if recycleBin == "" {
		if err := os.RemoveAll(path); err != nil {
			return "", fmt.Errorf("recyclebin: permanent delete folder: %w", err)
		}
		return "", nil
	}
	dest := filepath.Join(recycleBin, filepath.Base(path))
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("recyclebin: mkdir: %w", err)
	}
	if err := os.Rename(path, dest); err != nil {
		// Cross-filesystem fallback not implemented for folders; would require
		// a recursive copy. Callers on such setups should keep recycle_bin on
		// the same filesystem as the library (Sonarr's documented guidance).
		return "", fmt.Errorf("recyclebin: move folder: %w", err)
	}
	now := time.Now()
	_ = os.Chtimes(dest, now, now)
	return dest, nil
}

// Cleanup removes entries under recycleBin whose mtime is older than maxAge.
// Called by the scheduled CleanUpRecycleBin task. A zero-duration maxAge means
// "never purge" and is a no-op. Returns the count of removed entries.
func Cleanup(recycleBin string, maxAge time.Duration) (int, error) {
	if recycleBin == "" || maxAge <= 0 {
		return 0, nil
	}
	info, err := os.Stat(recycleBin)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("recyclebin: stat: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("recyclebin: %q is not a directory", recycleBin)
	}

	cutoff := time.Now().Add(-maxAge)
	entries, err := os.ReadDir(recycleBin)
	if err != nil {
		return 0, fmt.Errorf("recyclebin: readdir: %w", err)
	}
	var removed int
	for _, e := range entries {
		full := filepath.Join(recycleBin, e.Name())
		st, err := e.Info()
		if err != nil {
			continue
		}
		if st.ModTime().After(cutoff) {
			continue
		}
		if err := os.RemoveAll(full); err != nil {
			return removed, fmt.Errorf("recyclebin: remove %q: %w", full, err)
		}
		removed++
	}
	return removed, nil
}

// Empty removes every entry under recycleBin regardless of age.
// The user-facing "Empty Recycle Bin" command maps to this.
func Empty(recycleBin string) error {
	if recycleBin == "" {
		return nil
	}
	entries, err := os.ReadDir(recycleBin)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("recyclebin: readdir: %w", err)
	}
	for _, e := range entries {
		full := filepath.Join(recycleBin, e.Name())
		if err := os.RemoveAll(full); err != nil {
			return fmt.Errorf("recyclebin: remove %q: %w", full, err)
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
