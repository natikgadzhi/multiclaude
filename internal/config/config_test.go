package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		content    string // empty means no file
		wantErr    bool
		checkFunc  func(t *testing.T, cfg *Config)
	}{
		{
			name:    "missing file returns defaults",
			content: "",
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.DefaultProfile != "" {
					t.Errorf("DefaultProfile = %q, want empty", cfg.DefaultProfile)
				}
				// ClaudeHome stays as unexpanded default (~/.claude)
				// because we skip expansion when file is missing.
				if cfg.ClaudeHome != "~/.claude" {
					t.Errorf("ClaudeHome = %q, want %q", cfg.ClaudeHome, "~/.claude")
				}
			},
		},
		{
			name: "valid config loads all fields",
			content: `default_profile = "work"
claude_home = "/tmp/test-claude"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.DefaultProfile != "work" {
					t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "work")
				}
				if cfg.ClaudeHome != "/tmp/test-claude" {
					t.Errorf("ClaudeHome = %q, want %q", cfg.ClaudeHome, "/tmp/test-claude")
				}
			},
		},
		{
			name: "partial config applies defaults for missing fields",
			content: `default_profile = "personal"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				if cfg.DefaultProfile != "personal" {
					t.Errorf("DefaultProfile = %q, want %q", cfg.DefaultProfile, "personal")
				}
				// ClaudeHome should be expanded from default.
				home, _ := os.UserHomeDir()
				want := filepath.Join(home, ".claude")
				if cfg.ClaudeHome != want {
					t.Errorf("ClaudeHome = %q, want %q", cfg.ClaudeHome, want)
				}
			},
		},
		{
			name:    "invalid TOML returns error",
			content: `this is not valid [[ toml`,
			wantErr: true,
		},
		{
			name: "tilde expansion in claude_home",
			content: `claude_home = "~/.claude-custom"
`,
			wantErr: false,
			checkFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				home, _ := os.UserHomeDir()
				want := filepath.Join(home, ".claude-custom")
				if cfg.ClaudeHome != want {
					t.Errorf("ClaudeHome = %q, want %q", cfg.ClaudeHome, want)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.toml")

			if tt.content != "" {
				if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
					t.Fatalf("writing test config: %v", err)
				}
			}

			cfg, err := Load(path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.checkFunc != nil {
				tt.checkFunc(t, cfg)
			}
		})
	}
}

func TestProfilesDir(t *testing.T) {
	dir, err := ProfilesDir()
	if err != nil {
		t.Fatalf("ProfilesDir() error: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat ProfilesDir: %v", err)
	}
	if !info.IsDir() {
		t.Errorf("ProfilesDir %q is not a directory", dir)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "multiclaude", "profiles")
	if dir != want {
		t.Errorf("ProfilesDir() = %q, want %q", dir, want)
	}
}

