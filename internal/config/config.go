// Package config loads multiclaude's own TOML configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	cliconfig "github.com/natikgadzhi/cli-kit/config"
)

// Config holds multiclaude's application-level settings.
type Config struct {
	DefaultProfile string `toml:"default_profile"`
	ClaudeHome     string `toml:"claude_home"` // default: ~/.claude
	AutoBackup     bool   `toml:"auto_backup"` // default: true
}

// defaults returns a Config populated with default values.
func defaults() *Config {
	return &Config{
		ClaudeHome: "~/.claude",
		AutoBackup: true,
	}
}

// Load reads the TOML config file at path and returns a Config.
// If the file does not exist, Load returns a Config with default values and no error.
func Load(path string) (*Config, error) {
	expanded, err := cliconfig.ExpandTilde(path)
	if err != nil {
		return nil, err
	}

	cfg := defaults()
	if err := cliconfig.Load(expanded, cfg); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, err
	}

	// Apply defaults for fields not set in the file.
	if cfg.ClaudeHome == "" {
		cfg.ClaudeHome = "~/.claude"
	}

	// Expand tilde in ClaudeHome.
	cfg.ClaudeHome, err = cliconfig.ExpandTilde(cfg.ClaudeHome)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// configDir returns ~/.config/multiclaude, creating it if needed.
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "multiclaude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ProfilesDir returns the path to ~/.config/multiclaude/profiles,
// creating the directory if it does not exist.
func ProfilesDir() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// BackupsDir returns the path to ~/.config/multiclaude/backups,
// creating the directory if it does not exist.
func BackupsDir() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// ActiveFilePath returns the path to ~/.config/multiclaude/active,
// which stores the name of the currently active profile.
func ActiveFilePath() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "active"), nil
}

// ReadActiveProfile reads the name of the currently active profile
// from the active state file. Returns "" if the file doesn't exist.
func ReadActiveProfile() (string, error) {
	path, err := ActiveFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading active profile: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// WriteActiveProfile writes the name of the active profile to the state file.
// Pass an empty string to clear the active profile.
func WriteActiveProfile(name string) error {
	path, err := ActiveFilePath()
	if err != nil {
		return err
	}
	if name == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("clearing active profile: %w", err)
		}
		return nil
	}
	return os.WriteFile(path, []byte(name+"\n"), 0o644)
}

// ConfigFilePath returns the path to the config file, ensuring the
// config directory exists.
func ConfigFilePath() (string, error) {
	base, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "config.toml"), nil
}

// Save writes the Config to the given path in TOML format.
func Save(path string, cfg *Config) error {
	expanded, err := cliconfig.ExpandTilde(path)
	if err != nil {
		return err
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(expanded), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(expanded, buf.Bytes(), 0o644)
}
