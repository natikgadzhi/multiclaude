// Package claude provides read/write access to Claude Code's home directory
// and credentials. Credentials are stored in the OS keychain (not on disk)
// under the service "Claude Code-credentials".
package claude

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

// Keychain abstracts read/write/delete of a single keychain entry.
// The default implementation uses /usr/bin/security on macOS.
// Tests can inject an alternative (e.g. go-keyring mock).
type Keychain interface {
	Get(service, account string) (string, error)
	Set(service, account, value string) error
	Delete(service, account string) error
}

// ClaudeHome represents a Claude Code home directory (typically ~/.claude).
type ClaudeHome struct {
	Path     string
	Keychain Keychain // nil means use the default macOS security keychain
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
//
// When Keychain is nil, uses /usr/bin/security directly instead of go-keyring
// to avoid the base64 prefix that go-keyring adds on Set(). Handles backwards
// compatibility with entries previously written by go-keyring.
func (ch *ClaudeHome) ReadCredentials() ([]byte, error) {
	account := claudeKeychainAccount()

	var secret string
	if ch.Keychain != nil {
		s, err := ch.Keychain.Get(claudeKeychainService, account)
		if err != nil {
			return nil, fmt.Errorf("reading credentials from keychain: %w", err)
		}
		secret = s
	} else {
		s, err := securityGet(claudeKeychainService, account)
		if err != nil {
			return nil, err
		}
		secret = s
	}

	return decodeKeychainValue(secret)
}

// WriteCredentials writes credentials to Claude Code's OS keychain entry.
//
// When Keychain is nil, uses /usr/bin/security directly to write raw JSON
// without the base64 prefix that go-keyring would add. This ensures Claude
// Code (TypeScript/Node) can read the keychain value directly.
func (ch *ClaudeHome) WriteCredentials(data []byte) error {
	account := claudeKeychainAccount()

	if ch.Keychain != nil {
		return ch.Keychain.Set(claudeKeychainService, account, string(data))
	}
	return securitySet(claudeKeychainService, account, string(data))
}

// decodeKeychainValue handles backwards compatibility with go-keyring encoded values.
// go-keyring on macOS wraps all values with a "go-keyring-base64:" or "go-keyring-encoded:"
// prefix. If present, the prefix is stripped and the value is decoded.
func decodeKeychainValue(secret string) ([]byte, error) {
	const base64Prefix = "go-keyring-base64:"
	const hexPrefix = "go-keyring-encoded:"
	switch {
	case strings.HasPrefix(secret, base64Prefix):
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, base64Prefix))
		if err != nil {
			return nil, fmt.Errorf("decoding base64 credentials: %w", err)
		}
		return decoded, nil
	case strings.HasPrefix(secret, hexPrefix):
		decoded, err := hex.DecodeString(strings.TrimPrefix(secret, hexPrefix))
		if err != nil {
			return nil, fmt.Errorf("decoding hex credentials: %w", err)
		}
		return decoded, nil
	default:
		return []byte(secret), nil
	}
}

// securityGet reads a value from the macOS keychain using /usr/bin/security.
func securityGet(service, account string) (string, error) {
	cmd := exec.Command("/usr/bin/security", "find-generic-password",
		"-s", service, "-wa", account)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("reading credentials from keychain: %s", strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

// securitySet writes a raw value to the macOS keychain using /usr/bin/security.
// The value is written without any encoding prefix, so non-Go consumers
// (like Claude Code's TypeScript runtime) can read it directly.
func securitySet(service, account, value string) error {
	// Delete existing entry first (ignore "not found" errors).
	delCmd := exec.Command("/usr/bin/security", "delete-generic-password",
		"-s", service, "-a", account)
	_ = delCmd.Run()

	// Add via security -i with stdin to avoid shell escaping issues.
	addCmd := exec.Command("/usr/bin/security", "-i")
	stdin, err := addCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("creating stdin pipe: %w", err)
	}

	if err := addCmd.Start(); err != nil {
		return fmt.Errorf("starting security command: %w", err)
	}

	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n",
		shellQuote(service), shellQuote(account), shellQuote(value))
	if _, err := io.WriteString(stdin, command); err != nil {
		return fmt.Errorf("writing to security stdin: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("closing security stdin: %w", err)
	}

	if err := addCmd.Wait(); err != nil {
		return fmt.Errorf("writing credentials to keychain: %w", err)
	}
	return nil
}

// shellQuote wraps a string in single quotes for safe use in shell commands.
// Single quotes within the string are escaped as '"'"'.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
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
