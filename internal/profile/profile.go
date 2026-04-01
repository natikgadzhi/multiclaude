// Package profile provides CRUD operations for multiclaude profiles.
// Each profile represents a Claude Code account with credentials stored
// in the OS keychain and metadata/settings stored on disk.
package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/keychain"
)

// Profile represents a named Claude Code account profile.
type Profile struct {
	Name      string
	Email     string
	CreatedAt time.Time
	IsActive  bool
}

// metadata is the on-disk representation of profile metadata.
type metadata struct {
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages profile directories and their associated keychain entries.
type Store struct {
	profilesDir string
	claudeHome  *claude.ClaudeHome
	activeFile  string // path to the active profile state file (may be empty)
}

// NewStore creates a Store that manages profiles under profilesDir and
// checks active status against the given ClaudeHome.
func NewStore(profilesDir string, claudeHome *claude.ClaudeHome) *Store {
	return &Store{
		profilesDir: profilesDir,
		claudeHome:  claudeHome,
	}
}

// SetActiveFile sets the path to the active profile state file.
// When set, ActiveProfileName checks this file before falling back
// to symlink detection.
func (s *Store) SetActiveFile(path string) {
	s.activeFile = path
}

// WriteActive writes the given profile name to the active state file.
// Returns an error if no active file path has been configured.
func (s *Store) WriteActive(name string) error {
	if s.activeFile == "" {
		return fmt.Errorf("no active file path configured")
	}
	if name == "" {
		if err := os.Remove(s.activeFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clearing active profile: %w", err)
		}
		return nil
	}
	return os.WriteFile(s.activeFile, []byte(name+"\n"), 0o644)
}

// SaveState saves the current ClaudeHome credentials and settings into
// the given profile's directory. Used for auto-saving state before switching.
func (s *Store) SaveState(name string) error {
	dir := s.ProfileDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Save credentials to keychain.
	creds, err := s.claudeHome.ReadCredentials()
	if err == nil {
		// Best-effort: update keychain with latest credentials.
		_ = keychain.StoreCredentials(name, creds)
	}

	// Save settings snapshot.
	settings, err := s.claudeHome.ReadSettings()
	if err == nil {
		settingsBytes, err := json.MarshalIndent(settings, "", "  ")
		if err == nil {
			settingsBytes = append(settingsBytes, '\n')
			_ = os.WriteFile(filepath.Join(dir, "settings.json"), settingsBytes, 0o644)
		}
	}

	return nil
}

// RestoreState writes a profile's credentials and settings into ClaudeHome.
func (s *Store) RestoreState(name string) error {
	dir := s.ProfileDir(name)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Ensure ClaudeHome directory exists.
	if err := os.MkdirAll(s.claudeHome.Path, 0o755); err != nil {
		return fmt.Errorf("creating claude home: %w", err)
	}

	// Restore credentials from keychain.
	creds, err := keychain.GetCredentials(name)
	if err != nil {
		return fmt.Errorf("retrieving credentials for %q: %w", name, err)
	}
	if err := s.claudeHome.WriteCredentials(creds); err != nil {
		return fmt.Errorf("writing credentials: %w", err)
	}

	// Restore settings from profile snapshot.
	settingsBytes, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		// Settings may not exist; that's okay for older profiles.
		return nil
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return fmt.Errorf("parsing profile settings: %w", err)
	}
	if err := s.claudeHome.WriteSettings(settings); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	return nil
}

// ProfileDir returns the filesystem path for a named profile's directory.
func (s *Store) ProfileDir(name string) string {
	return filepath.Join(s.profilesDir, name)
}

// Create builds a new profile with the given name, storing credentials in the
// keychain and writing metadata and settings snapshots to disk.
// The email parameter is recorded in metadata.json; the raw creds go only to
// the keychain and are never persisted on disk.
func (s *Store) Create(name string, creds []byte, settings map[string]any, email string) error {
	dir := s.ProfileDir(name)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating profile directory: %w", err)
	}

	// Write metadata.json.
	meta := metadata{
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}
	metaBytes = append(metaBytes, '\n')
	if err := os.WriteFile(filepath.Join(dir, "metadata.json"), metaBytes, 0o644); err != nil {
		return fmt.Errorf("writing metadata: %w", err)
	}

	// Write settings.json snapshot.
	settingsBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	settingsBytes = append(settingsBytes, '\n')
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), settingsBytes, 0o644); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	// Store credentials in the keychain only.
	if err := keychain.StoreCredentials(name, creds); err != nil {
		// Clean up the directory on keychain failure so we don't leave a partial profile.
		os.RemoveAll(dir)
		return fmt.Errorf("storing credentials in keychain: %w", err)
	}

	return nil
}

// Get reads a single profile by name.
// It returns an error if the profile directory or metadata is missing.
func (s *Store) Get(name string) (*Profile, error) {
	dir := s.ProfileDir(name)

	metaBytes, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		return nil, fmt.Errorf("reading profile metadata: %w", err)
	}

	var meta metadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("parsing profile metadata: %w", err)
	}

	active, _ := s.ActiveProfileName()

	return &Profile{
		Name:      name,
		Email:     meta.Email,
		CreatedAt: meta.CreatedAt,
		IsActive:  name == active,
	}, nil
}

// List returns all profiles found in the profiles directory.
// The active profile (if any) is marked with IsActive = true.
func (s *Store) List() ([]Profile, error) {
	entries, err := os.ReadDir(s.profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	active, _ := s.ActiveProfileName()

	var profiles []Profile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		metaPath := filepath.Join(s.profilesDir, name, "metadata.json")
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			// Skip directories without metadata (not valid profiles).
			continue
		}

		var meta metadata
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			continue
		}

		profiles = append(profiles, Profile{
			Name:      name,
			Email:     meta.Email,
			CreatedAt: meta.CreatedAt,
			IsActive:  name == active,
		})
	}

	return profiles, nil
}

// Delete removes a profile's directory and its keychain entry.
func (s *Store) Delete(name string) error {
	dir := s.ProfileDir(name)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Remove keychain entry. Ignore errors if the entry doesn't exist
	// (the profile may have been partially created).
	_ = keychain.DeleteCredentials(name)

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing profile directory: %w", err)
	}

	return nil
}

// Rename moves a profile from old name to new name, including its keychain entry.
func (s *Store) Rename(old, new string) error {
	oldDir := s.ProfileDir(old)
	newDir := s.ProfileDir(new)

	if _, err := os.Stat(oldDir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q does not exist", old)
	}
	if _, err := os.Stat(newDir); err == nil {
		return fmt.Errorf("profile %q already exists", new)
	}

	// Move the keychain entry: read old, store under new name, delete old.
	creds, err := keychain.GetCredentials(old)
	if err != nil {
		return fmt.Errorf("reading keychain for rename: %w", err)
	}
	if err := keychain.StoreCredentials(new, creds); err != nil {
		return fmt.Errorf("storing renamed credentials: %w", err)
	}
	_ = keychain.DeleteCredentials(old)

	// Rename the directory.
	if err := os.Rename(oldDir, newDir); err != nil {
		// Attempt to roll back keychain changes on failure.
		_ = keychain.StoreCredentials(old, creds)
		_ = keychain.DeleteCredentials(new)
		return fmt.Errorf("renaming profile directory: %w", err)
	}

	return nil
}

// Exists returns true if a profile directory with the given name exists.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.ProfileDir(name))
	return err == nil
}

// ActiveProfileName returns the name of the currently active profile.
// It first checks the state file (~/.config/multiclaude/active), then
// falls back to resolving the ClaudeHome symlink for backwards compatibility.
// Returns "" with no error if no profile is active.
func (s *Store) ActiveProfileName() (string, error) {
	// Check the state file first.
	if s.activeFile != "" {
		data, err := os.ReadFile(s.activeFile)
		if err == nil {
			name := strings.TrimSpace(string(data))
			if name != "" && s.Exists(name) {
				return name, nil
			}
		}
	}

	// Fall back to symlink detection.
	if !s.claudeHome.IsSymlink() {
		return "", nil
	}

	target, err := s.claudeHome.SymlinkTarget()
	if err != nil {
		return "", fmt.Errorf("resolving active profile: %w", err)
	}

	// The symlink target should be an absolute path under profilesDir.
	// Extract the profile name from it.
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", fmt.Errorf("resolving absolute path: %w", err)
	}

	absProfiles, err := filepath.Abs(s.profilesDir)
	if err != nil {
		return "", fmt.Errorf("resolving profiles dir: %w", err)
	}

	rel, err := filepath.Rel(absProfiles, absTarget)
	if err != nil {
		return "", fmt.Errorf("computing relative path: %w", err)
	}

	// The relative path should be a single directory name (the profile name).
	// If it contains separators or "..", the symlink doesn't point into our profiles dir.
	if rel == ".." || filepath.Base(rel) != rel {
		return "", nil
	}

	return rel, nil
}
