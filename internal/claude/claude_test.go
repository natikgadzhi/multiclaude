package claude

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// mockKeychain is a simple in-memory Keychain for tests.
type mockKeychain struct {
	mu    sync.Mutex
	store map[string]string // key = "service\x00account"
}

func newMockKeychain() *mockKeychain {
	return &mockKeychain{store: make(map[string]string)}
}

func (m *mockKeychain) key(service, account string) string {
	return service + "\x00" + account
}

func (m *mockKeychain) Get(service, account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.store[m.key(service, account)]
	if !ok {
		return "", fmt.Errorf("secret not found in keychain")
	}
	return v, nil
}

func (m *mockKeychain) Set(service, account, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[m.key(service, account)] = value
	return nil
}

func (m *mockKeychain) Delete(service, account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.store, m.key(service, account))
	return nil
}

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

	mock := newMockKeychain()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), fakeSettings(), 0644); err != nil {
		t.Fatal(err)
	}

	// Put credentials in the mock keychain (same service Claude Code uses).
	creds := fakeCredentials(email)
	if err := mock.Set(claudeKeychainService, claudeKeychainAccount(), string(creds)); err != nil {
		t.Fatal(err)
	}

	ch := NewClaudeHome(dir)
	ch.Keychain = mock
	return ch
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
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = newMockKeychain()

	_, err := ch.ReadCredentials()
	if err == nil {
		t.Fatal("ReadCredentials() expected error for missing keychain entry, got nil")
	}
}

func TestWriteCredentials(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = newMockKeychain()

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

func TestReadCredentials_decodesGoKeyringBase64(t *testing.T) {
	mock := newMockKeychain()
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = mock

	// Simulate a value that was written by go-keyring (with base64 prefix).
	original := fakeCredentials("alice@example.com")
	encoded := "go-keyring-base64:" + base64.StdEncoding.EncodeToString(original)
	mock.Set(claudeKeychainService, claudeKeychainAccount(), encoded)

	got, err := ch.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials() error: %v", err)
	}
	if string(got) != string(original) {
		t.Errorf("go-keyring-base64 decode mismatch:\ngot:  %s\nwant: %s", got, original)
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

func TestActiveAccountInfo(t *testing.T) {
	mock := newMockKeychain()
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = mock

	creds := fakeCredentials("alice@example.com")
	mock.Set(claudeKeychainService, claudeKeychainAccount(), string(creds))

	info, err := ch.ActiveAccountInfo()
	if err != nil {
		t.Fatalf("ActiveAccountInfo() error: %v", err)
	}
	if info.Email != "alice@example.com" {
		t.Errorf("Email = %q, want %q", info.Email, "alice@example.com")
	}
	if info.Label() != "alice@example.com" {
		t.Errorf("Label() = %q, want %q", info.Label(), "alice@example.com")
	}
}

func TestActiveAccountInfo_noEmail(t *testing.T) {
	mock := newMockKeychain()
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = mock

	creds := fakeCredentials("")
	mock.Set(claudeKeychainService, claudeKeychainAccount(), string(creds))

	info, err := ch.ActiveAccountInfo()
	if err != nil {
		t.Fatalf("ActiveAccountInfo() error: %v", err)
	}
	// Label falls back to OS username when no email
	if info.Label() == "" {
		t.Error("Label() should not be empty")
	}
}

func TestActiveAccountInfo_noKeychain(t *testing.T) {
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = newMockKeychain()

	_, err := ch.ActiveAccountInfo()
	if err == nil {
		t.Fatal("expected error for missing keychain, got nil")
	}
}

func TestActiveAccountInfo_malformedJSON(t *testing.T) {
	mock := newMockKeychain()
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = mock
	mock.Set(claudeKeychainService, claudeKeychainAccount(), "not json")

	_, err := ch.ActiveAccountInfo()
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestActiveAccountInfo_missingOAuthKey(t *testing.T) {
	mock := newMockKeychain()
	ch := NewClaudeHome(t.TempDir())
	ch.Keychain = mock
	mock.Set(claudeKeychainService, claudeKeychainAccount(), `{"otherKey": {}}`)

	_, err := ch.ActiveAccountInfo()
	if err == nil {
		t.Fatal("expected error for missing claudeAiOauth key, got nil")
	}
}
