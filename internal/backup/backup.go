// Package backup provides create/list/restore/delete operations for
// multiclaude profile snapshots. Each backup captures all profile directories
// and their associated keychain credentials.
package backup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/natikgadzhi/multiclaude/internal/keychain"
	"github.com/natikgadzhi/multiclaude/internal/profile"
)

// Backup describes a saved snapshot.
type Backup struct {
	Name      string
	CreatedAt time.Time
	Profiles  int // number of profiles in the backup
}

// backupMetadata is the on-disk JSON representation stored in each backup.
type backupMetadata struct {
	CreatedAt time.Time `json:"created_at"`
	Profiles  int       `json:"profiles"`
}

// Manager manages backup lifecycle operations.
type Manager struct {
	backupsDir string
	store      *profile.Store
}

// NewManager creates a Manager that stores backups under backupsDir and
// reads/writes profiles via the given Store.
func NewManager(backupsDir string, store *profile.Store) *Manager {
	return &Manager{
		backupsDir: backupsDir,
		store:      store,
	}
}

// Create snapshots all profiles and their keychain credentials into a
// new backup named name.
func (m *Manager) Create(name string) error {
	backupDir := filepath.Join(m.backupsDir, name)
	if _, err := os.Stat(backupDir); err == nil {
		return fmt.Errorf("backup %q already exists", name)
	}

	profiles, err := m.store.List()
	if err != nil {
		return fmt.Errorf("listing profiles: %w", err)
	}

	profilesDst := filepath.Join(backupDir, "profiles")
	if err := os.MkdirAll(profilesDst, 0o755); err != nil {
		return fmt.Errorf("creating backup profiles dir: %w", err)
	}

	keychainDst := filepath.Join(backupDir, "keychain")
	if err := os.MkdirAll(keychainDst, 0o755); err != nil {
		return fmt.Errorf("creating backup keychain dir: %w", err)
	}

	// Copy each profile directory and export its keychain credentials.
	for _, p := range profiles {
		srcDir := m.store.ProfileDir(p.Name)
		dstDir := filepath.Join(profilesDst, p.Name)
		if err := copyDir(srcDir, dstDir); err != nil {
			// Clean up partial backup on failure.
			os.RemoveAll(backupDir)
			return fmt.Errorf("copying profile %q: %w", p.Name, err)
		}

		// Export keychain credentials as base64 file.
		creds, err := keychain.GetCredentials(p.Name)
		if err != nil {
			// Credential may be missing; warn but continue.
			continue
		}
		encoded := base64.StdEncoding.EncodeToString(creds)
		credFile := filepath.Join(keychainDst, p.Name+".b64")
		if err := os.WriteFile(credFile, []byte(encoded), 0o600); err != nil {
			os.RemoveAll(backupDir)
			return fmt.Errorf("writing keychain backup for %q: %w", p.Name, err)
		}
	}

	// Write metadata.json.
	meta := backupMetadata{
		CreatedAt: time.Now().UTC(),
		Profiles:  len(profiles),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		os.RemoveAll(backupDir)
		return fmt.Errorf("marshaling backup metadata: %w", err)
	}
	metaBytes = append(metaBytes, '\n')
	if err := os.WriteFile(filepath.Join(backupDir, "metadata.json"), metaBytes, 0o644); err != nil {
		os.RemoveAll(backupDir)
		return fmt.Errorf("writing backup metadata: %w", err)
	}

	return nil
}

// List returns all available backups, sorted by creation time (newest first).
func (m *Manager) List() ([]Backup, error) {
	entries, err := os.ReadDir(m.backupsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading backups directory: %w", err)
	}

	var backups []Backup
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := filepath.Join(m.backupsDir, entry.Name(), "metadata.json")
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			// Skip directories without metadata.
			continue
		}

		var meta backupMetadata
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			continue
		}

		backups = append(backups, Backup{
			Name:      entry.Name(),
			CreatedAt: meta.CreatedAt,
			Profiles:  meta.Profiles,
		})
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// Restore replaces current profiles and keychain entries with those from the
// named backup. Existing profiles are removed first.
func (m *Manager) Restore(name string) error {
	backupDir := filepath.Join(m.backupsDir, name)
	metaPath := filepath.Join(backupDir, "metadata.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("backup %q does not exist", name)
	}

	profilesSrc := filepath.Join(backupDir, "profiles")
	keychainSrc := filepath.Join(backupDir, "keychain")

	// Delete all current profiles first (dirs + keychain entries).
	existing, err := m.store.List()
	if err != nil {
		return fmt.Errorf("listing current profiles: %w", err)
	}
	for _, p := range existing {
		if err := m.store.Delete(p.Name); err != nil {
			return fmt.Errorf("removing existing profile %q: %w", p.Name, err)
		}
	}

	// Read backup profile directories.
	entries, err := os.ReadDir(profilesSrc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // backup had no profiles
		}
		return fmt.Errorf("reading backup profiles: %w", err)
	}

	profilesDir := m.store.ProfilesDir()
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pName := entry.Name()
		src := filepath.Join(profilesSrc, pName)
		dst := filepath.Join(profilesDir, pName)
		if err := copyDir(src, dst); err != nil {
			return fmt.Errorf("restoring profile %q: %w", pName, err)
		}

		// Restore keychain credentials.
		credFile := filepath.Join(keychainSrc, pName+".b64")
		encoded, err := os.ReadFile(credFile)
		if err != nil {
			// Credential file may not exist; skip.
			continue
		}
		creds, err := base64.StdEncoding.DecodeString(string(encoded))
		if err != nil {
			return fmt.Errorf("decoding credentials for %q: %w", pName, err)
		}
		if err := keychain.StoreCredentials(pName, creds); err != nil {
			return fmt.Errorf("restoring keychain for %q: %w", pName, err)
		}
	}

	return nil
}

// PruneAutoBackups deletes the oldest auto-* backups beyond the keep limit.
// Auto-backups are identified by the "auto-" prefix in their name.
func (m *Manager) PruneAutoBackups(keep int) error {
	backups, err := m.List()
	if err != nil {
		return fmt.Errorf("listing backups for prune: %w", err)
	}

	// Filter to only auto-backups. List() returns newest first.
	var autoBackups []Backup
	for _, b := range backups {
		if len(b.Name) > 5 && b.Name[:5] == "auto-" {
			autoBackups = append(autoBackups, b)
		}
	}

	if len(autoBackups) <= keep {
		return nil
	}

	// Delete oldest beyond the keep limit (list is newest-first).
	for _, b := range autoBackups[keep:] {
		if err := m.Delete(b.Name); err != nil {
			return fmt.Errorf("pruning auto-backup %q: %w", b.Name, err)
		}
	}

	return nil
}

// Delete removes a backup by name.
func (m *Manager) Delete(name string) error {
	backupDir := filepath.Join(m.backupsDir, name)
	metaPath := filepath.Join(backupDir, "metadata.json")
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return fmt.Errorf("backup %q does not exist", name)
	}

	if err := os.RemoveAll(backupDir); err != nil {
		return fmt.Errorf("removing backup directory: %w", err)
	}
	return nil
}

// copyDir recursively copies the directory tree rooted at src to dst.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
