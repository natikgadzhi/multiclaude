// Package claude provides read/write access to Claude Code's home directory
// and credentials. Credentials are stored in the OS keychain (not on disk)
// under the service "Claude Code-credentials".
package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

// ClaudeHome represents a Claude Code home directory (typically ~/.claude).
type ClaudeHome struct {
	Path string
}

// NewClaudeHome creates a ClaudeHome pointing at the given directory path.
func NewClaudeHome(path string) *ClaudeHome {
	return &ClaudeHome{Path: path}
}

const (
	// claudeKeychainService is the keychain service used by Claude Code itself.
	claudeKeychainService = "Claude Code-credentials"
)

// claudeKeychainAccount returns the keychain account Claude Code uses.
// Claude Code stores credentials under the OS username.
func claudeKeychainAccount() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	// Fallback: derive from home directory.
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Base(home)
	}
	return ""
}

// settingsPath returns the full path to settings.json.
func (ch *ClaudeHome) settingsPath() string {
	return filepath.Join(ch.Path, "settings.json")
}

// Exists returns true if the Claude home directory exists (as a real dir or symlink).
func (ch *ClaudeHome) Exists() bool {
	_, err := os.Lstat(ch.Path)
	return err == nil
}

// IsSymlink returns true if the Claude home path is a symbolic link.
func (ch *ClaudeHome) IsSymlink() bool {
	fi, err := os.Lstat(ch.Path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}

// SymlinkTarget returns the target of the symlink at the Claude home path.
// Returns an error if the path is not a symlink or cannot be read.
func (ch *ClaudeHome) SymlinkTarget() (string, error) {
	target, err := os.Readlink(ch.Path)
	if err != nil {
		return "", fmt.Errorf("reading symlink target: %w", err)
	}
	return target, nil
}

// ReadCredentials reads Claude Code's credentials from the OS keychain.
// Returns the raw JSON bytes (the entire credential object).
func (ch *ClaudeHome) ReadCredentials() ([]byte, error) {
	secret, err := keyring.Get(claudeKeychainService, claudeKeychainAccount())
	if err != nil {
		return nil, fmt.Errorf("reading credentials from keychain: %w", err)
	}
	return []byte(secret), nil
}

// WriteCredentials writes credentials to Claude Code's OS keychain entry.
func (ch *ClaudeHome) WriteCredentials(data []byte) error {
	if err := keyring.Set(claudeKeychainService, claudeKeychainAccount(), string(data)); err != nil {
		return fmt.Errorf("writing credentials to keychain: %w", err)
	}
	return nil
}

// ReadSettings parses settings.json into a generic map.
func (ch *ClaudeHome) ReadSettings() (map[string]any, error) {
	data, err := os.ReadFile(ch.settingsPath())
	if err != nil {
		return nil, fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}
	return settings, nil
}

// WriteSettings serializes the settings map and writes it to settings.json.
// The file is written atomically via a temporary file and rename.
func (ch *ClaudeHome) WriteSettings(settings map[string]any) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}
	// Append a trailing newline for POSIX compliance.
	data = append(data, '\n')
	return atomicWrite(ch.settingsPath(), data, 0644)
}

// AccountInfo holds identifying information extracted from Claude Code credentials.
type AccountInfo struct {
	Email            string // may be empty if not present in credentials
	SubscriptionType string // e.g. "individual", "team"
	RateLimitTier    string // e.g. "free", "pro"
}

// Label returns a human-readable label for the account.
// Uses email if available, otherwise subscription type, otherwise OS username.
func (a *AccountInfo) Label() string {
	if a.Email != "" {
		return a.Email
	}
	if a.SubscriptionType != "" {
		return a.SubscriptionType
	}
	return claudeKeychainAccount()
}

// ActiveAccountInfo extracts account information from the Claude Code credentials.
func (ch *ClaudeHome) ActiveAccountInfo() (*AccountInfo, error) {
	data, err := ch.ReadCredentials()
	if err != nil {
		return nil, err
	}
	return extractAccountInfo(data)
}

func extractAccountInfo(data []byte) (*AccountInfo, error) {
	var creds map[string]json.RawMessage
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}

	oauthRaw, ok := creds["claudeAiOauth"]
	if !ok {
		return nil, fmt.Errorf("credentials missing claudeAiOauth key")
	}

	var oauth struct {
		Email            string `json:"email"`
		SubscriptionType string `json:"subscriptionType"`
		RateLimitTier    string `json:"rateLimitTier"`
	}
	if err := json.Unmarshal(oauthRaw, &oauth); err != nil {
		return nil, fmt.Errorf("parsing claudeAiOauth: %w", err)
	}

	return &AccountInfo{
		Email:            oauth.Email,
		SubscriptionType: oauth.SubscriptionType,
		RateLimitTier:    oauth.RateLimitTier,
	}, nil
}

// atomicWrite writes data to a temporary file in the same directory as path
// and then renames it into place, ensuring the write is atomic.
func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".multiclaude-tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up on any error path.
	defer func() {
		if tmpName != "" {
			os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("setting file permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	// Rename succeeded — disarm the deferred cleanup.
	tmpName = ""
	return nil
}
