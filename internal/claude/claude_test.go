package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// fakeCredentials returns a JSON byte slice that mimics a real .credentials.json.
func fakeCredentials(email string) []byte {
	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"token":        "fake-access-token-abc123",
			"expiresAt":    "2099-01-01T00:00:00.000Z",
			"refreshToken": "fake-refresh-token-xyz789",
			"email":        email,
		},
	}
	data, _ := json.MarshalIndent(creds, "", "  ")
	return data
}

// fakeSettings returns a JSON byte slice that mimics settings.json.
func fakeSettings() []byte {
	settings := map[string]any{
		"permissions": map[string]any{
			"allow": []string{"Bash(git *)"},
		},
		"model": "opus",
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	return data
}

// setupClaudeHome creates a temp directory with fake credential and settings files.
func setupClaudeHome(t *testing.T, email string) *ClaudeHome {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), fakeCredentials(email), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), fakeSettings(), 0644); err != nil {
		t.Fatal(err)
	}
	return NewClaudeHome(dir)
}

func TestNewClaudeHome(t *testing.T) {
	ch := NewClaudeHome("/some/path")
	if ch.Path != "/some/path" {
		t.Errorf("Path = %q, want %q", ch.Path, "/some/path")
	}
}

func TestExists(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T) *ClaudeHome
		exists bool
	}{
		{
			name: "existing directory",
			setup: func(t *testing.T) *ClaudeHome {
				return NewClaudeHome(t.TempDir())
			},
			exists: true,
		},
		{
			name: "nonexistent directory",
			setup: func(t *testing.T) *ClaudeHome {
				return NewClaudeHome(filepath.Join(t.TempDir(), "does-not-exist"))
			},
			exists: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := tt.setup(t)
			if got := ch.Exists(); got != tt.exists {
				t.Errorf("Exists() = %v, want %v", got, tt.exists)
			}
		})
	}
}

func TestReadCredentials(t *testing.T) {
	ch := setupClaudeHome(t, "alice@example.com")

	data, err := ch.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials() error: %v", err)
	}

	// Verify it round-trips as valid JSON with expected structure.
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("returned data is not valid JSON: %v", err)
	}
	oauth, ok := parsed["claudeAiOauth"].(map[string]any)
	if !ok {
		t.Fatal("missing claudeAiOauth key in credentials")
	}
	if oauth["email"] != "alice@example.com" {
		t.Errorf("email = %v, want %q", oauth["email"], "alice@example.com")
	}
}

func TestReadCredentials_missing(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ReadCredentials()
	if err == nil {
		t.Fatal("ReadCredentials() expected error for missing file, got nil")
	}
}

func TestWriteCredentials(t *testing.T) {
	dir := t.TempDir()
	ch := NewClaudeHome(dir)

	original := fakeCredentials("bob@example.com")
	if err := ch.WriteCredentials(original); err != nil {
		t.Fatalf("WriteCredentials() error: %v", err)
	}

	// Read it back and verify.
	got, err := ch.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials() error: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("round-trip mismatch:\ngot:  %s\nwant: %s", got, original)
	}

	// Verify file permissions are restricted.
	fi, err := os.Stat(filepath.Join(dir, ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	perm := fi.Mode().Perm()
	if perm != 0600 {
		t.Errorf("permissions = %o, want 0600", perm)
	}
}

func TestReadSettings(t *testing.T) {
	ch := setupClaudeHome(t, "alice@example.com")

	settings, err := ch.ReadSettings()
	if err != nil {
		t.Fatalf("ReadSettings() error: %v", err)
	}

	model, ok := settings["model"].(string)
	if !ok || model != "opus" {
		t.Errorf("model = %v, want %q", settings["model"], "opus")
	}

	perms, ok := settings["permissions"].(map[string]any)
	if !ok {
		t.Fatal("missing permissions key")
	}
	allow, ok := perms["allow"].([]any)
	if !ok || len(allow) == 0 {
		t.Fatal("missing or empty permissions.allow")
	}
}

func TestReadSettings_missing(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ReadSettings()
	if err == nil {
		t.Fatal("ReadSettings() expected error for missing file, got nil")
	}
}

func TestWriteSettings(t *testing.T) {
	dir := t.TempDir()
	ch := NewClaudeHome(dir)

	original := map[string]any{
		"model": "sonnet",
		"permissions": map[string]any{
			"allow": []any{"Bash(git *)"},
		},
	}
	if err := ch.WriteSettings(original); err != nil {
		t.Fatalf("WriteSettings() error: %v", err)
	}

	got, err := ch.ReadSettings()
	if err != nil {
		t.Fatalf("ReadSettings() error: %v", err)
	}
	if got["model"] != "sonnet" {
		t.Errorf("model = %v, want %q", got["model"], "sonnet")
	}
}

func TestActiveEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantEmail string
		wantErr   bool
	}{
		{
			name:      "valid email",
			email:     "alice@example.com",
			wantEmail: "alice@example.com",
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := setupClaudeHome(t, tt.email)
			got, err := ch.ActiveEmail()
			if tt.wantErr {
				if err == nil {
					t.Fatal("ActiveEmail() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ActiveEmail() error: %v", err)
			}
			if got != tt.wantEmail {
				t.Errorf("ActiveEmail() = %q, want %q", got, tt.wantEmail)
			}
		})
	}
}

func TestActiveEmail_missingFile(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for missing file, got nil")
	}
}

func TestActiveEmail_malformedJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), []byte("not json"), 0600); err != nil {
		t.Fatal(err)
	}
	ch := NewClaudeHome(dir)

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for malformed JSON, got nil")
	}
}

func TestActiveEmail_missingOAuthKey(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"someOtherKey": {"email": "test@test.com"}}`)
	if err := os.WriteFile(filepath.Join(dir, ".credentials.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	ch := NewClaudeHome(dir)

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for missing claudeAiOauth key, got nil")
	}
}

func TestIsSymlink(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) *ClaudeHome
		symlink bool
	}{
		{
			name: "regular directory",
			setup: func(t *testing.T) *ClaudeHome {
				return NewClaudeHome(t.TempDir())
			},
			symlink: false,
		},
		{
			name: "symlink",
			setup: func(t *testing.T) *ClaudeHome {
				target := t.TempDir()
				parent := t.TempDir()
				link := filepath.Join(parent, "claude-link")
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				return NewClaudeHome(link)
			},
			symlink: true,
		},
		{
			name: "nonexistent path",
			setup: func(t *testing.T) *ClaudeHome {
				return NewClaudeHome(filepath.Join(t.TempDir(), "nope"))
			},
			symlink: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := tt.setup(t)
			if got := ch.IsSymlink(); got != tt.symlink {
				t.Errorf("IsSymlink() = %v, want %v", got, tt.symlink)
			}
		})
	}
}

func TestSymlinkTarget(t *testing.T) {
	target := t.TempDir()
	parent := t.TempDir()
	link := filepath.Join(parent, "claude-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	ch := NewClaudeHome(link)

	got, err := ch.SymlinkTarget()
	if err != nil {
		t.Fatalf("SymlinkTarget() error: %v", err)
	}
	if got != target {
		t.Errorf("SymlinkTarget() = %q, want %q", got, target)
	}
}

func TestSymlinkTarget_notSymlink(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.SymlinkTarget()
	if err == nil {
		t.Fatal("SymlinkTarget() expected error for non-symlink, got nil")
	}
}

func TestAtomicWriteDoesNotLeaveTempFiles(t *testing.T) {
	dir := t.TempDir()
	ch := NewClaudeHome(dir)

	if err := ch.WriteCredentials(fakeCredentials("test@test.com")); err != nil {
		t.Fatal(err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != ".credentials.json" {
			t.Errorf("unexpected file left behind: %s", e.Name())
		}
	}
}
