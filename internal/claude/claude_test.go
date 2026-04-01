package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// fakeCredentials returns a JSON byte slice that mimics Claude Code's credential format.
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

// setupClaudeHome creates a temp directory with settings and puts credentials in mock keychain.
func setupClaudeHome(t *testing.T, email string) *ClaudeHome {
	t.Helper()
	keyring.MockInit()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), fakeSettings(), 0644); err != nil {
		t.Fatal(err)
	}

	// Put credentials in the mock keychain (same service Claude Code uses).
	creds := fakeCredentials(email)
	if err := keyring.Set(claudeKeychainService, claudeKeychainAccount(), string(creds)); err != nil {
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
			name: "non-existing directory",
			setup: func(t *testing.T) *ClaudeHome {
				return NewClaudeHome(filepath.Join(t.TempDir(), "nonexistent"))
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
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ReadCredentials()
	if err == nil {
		t.Fatal("ReadCredentials() expected error for missing keychain entry, got nil")
	}
}

func TestWriteCredentials(t *testing.T) {
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())

	original := fakeCredentials("bob@example.com")
	if err := ch.WriteCredentials(original); err != nil {
		t.Fatalf("WriteCredentials() error: %v", err)
	}

	// Read back from keychain.
	got, err := ch.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials() error: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("round-trip mismatch:\ngot:  %s\nwant: %s", got, original)
	}
}

func TestReadSettings(t *testing.T) {
	ch := setupClaudeHome(t, "alice@example.com")

	settings, err := ch.ReadSettings()
	if err != nil {
		t.Fatalf("ReadSettings() error: %v", err)
	}

	if settings["model"] != "opus" {
		t.Errorf("model = %v, want %q", settings["model"], "opus")
	}
}

func TestReadSettings_missing(t *testing.T) {
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ReadSettings()
	if err == nil {
		t.Fatal("ReadSettings() expected error for missing file, got nil")
	}
}

func TestWriteSettings(t *testing.T) {
	dir := t.TempDir()
	ch := NewClaudeHome(dir)

	settings := map[string]any{
		"model": "sonnet",
		"debug": true,
	}
	if err := ch.WriteSettings(settings); err != nil {
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

func TestIsSymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	link := filepath.Join(dir, "link")

	os.MkdirAll(real, 0755)
	os.Symlink(real, link)

	if NewClaudeHome(real).IsSymlink() {
		t.Error("real directory should not be a symlink")
	}
	if !NewClaudeHome(link).IsSymlink() {
		t.Error("symlink should be detected")
	}
}

func TestSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	link := filepath.Join(dir, "link")

	os.MkdirAll(real, 0755)
	os.Symlink(real, link)

	target, err := NewClaudeHome(link).SymlinkTarget()
	if err != nil {
		t.Fatalf("SymlinkTarget() error: %v", err)
	}
	if target != real {
		t.Errorf("target = %q, want %q", target, real)
	}
}

func TestActiveEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		wantEmail string
		wantErr   bool
	}{
		{"valid email", "alice@example.com", "alice@example.com", false},
		{"empty email returns unknown", "", "(unknown)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyring.MockInit()
			ch := NewClaudeHome(t.TempDir())

			creds := fakeCredentials(tt.email)
			keyring.Set(claudeKeychainService, claudeKeychainAccount(), string(creds))

			got, err := ch.ActiveEmail()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ActiveEmail() expected error, got nil")
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

func TestActiveEmail_noKeychain(t *testing.T) {
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for missing keychain, got nil")
	}
}

func TestActiveEmail_malformedJSON(t *testing.T) {
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())
	keyring.Set(claudeKeychainService, claudeKeychainAccount(), "not json")

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for malformed JSON, got nil")
	}
}

func TestActiveEmail_missingOAuthKey(t *testing.T) {
	keyring.MockInit()
	ch := NewClaudeHome(t.TempDir())
	keyring.Set(claudeKeychainService, claudeKeychainAccount(), `{"otherKey": {}}`)

	_, err := ch.ActiveEmail()
	if err == nil {
		t.Fatal("ActiveEmail() expected error for missing claudeAiOauth key, got nil")
	}
}
