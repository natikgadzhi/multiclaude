package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/keychain"
	keyringlib "github.com/zalando/go-keyring"
)

// goKeyringAdapter wraps go-keyring to implement claude.Keychain for tests.
type goKeyringAdapter struct{}

func (goKeyringAdapter) Get(service, account string) (string, error) {
	return keyringlib.Get(service, account)
}

func (goKeyringAdapter) Set(service, account, value string) error {
	return keyringlib.Set(service, account, value)
}

func (goKeyringAdapter) Delete(service, account string) error {
	return keyringlib.Delete(service, account)
}

// testCreds returns fake credential JSON bytes for testing.
func testCreds() []byte {
	return []byte(`{"claudeAiOauth":{"token":"test-token","email":"alice@example.com"}}`)
}

// testSettings returns a settings map for testing.
func testSettings() map[string]any {
	return map[string]any{
		"model": "opus",
		"permissions": map[string]any{
			"allow": []any{"Bash(git *)"},
		},
	}
}

// newTestStore creates a Store backed by a temp directory and a ClaudeHome
// in another temp directory. It also initializes the keyring mock.
func newTestStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	keyringlib.MockInit()

	profilesDir := filepath.Join(t.TempDir(), "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	ch := claude.NewClaudeHome(claudeDir)
	ch.Keychain = goKeyringAdapter{}

	return NewStore(profilesDir, ch), profilesDir, claudeDir
}

func TestCreate(t *testing.T) {
	store, profilesDir, _ := newTestStore(t)

	err := store.Create("work", testCreds(), testSettings(), CreateOptions{Email: "alice@example.com"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Profile directory should exist.
	dir := filepath.Join(profilesDir, "work")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatal("profile directory was not created")
	}

	// metadata.json should be readable and correct.
	metaBytes, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatalf("reading metadata.json: %v", err)
	}
	var meta metadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("parsing metadata.json: %v", err)
	}
	if meta.Email != "alice@example.com" {
		t.Errorf("metadata email = %q, want %q", meta.Email, "alice@example.com")
	}
	if meta.CreatedAt.IsZero() {
		t.Error("metadata created_at is zero")
	}

	// settings.json should be readable.
	settingsBytes, err := os.ReadFile(filepath.Join(dir, "settings.json"))
	if err != nil {
		t.Fatalf("reading settings.json: %v", err)
	}
	var settings map[string]any
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	if settings["model"] != "opus" {
		t.Errorf("settings model = %v, want %q", settings["model"], "opus")
	}

	// Keychain should have the credentials.
	if !keychain.HasCredentials("work") {
		t.Error("keychain does not have credentials for profile 'work'")
	}

	creds, err := keychain.GetCredentials("work")
	if err != nil {
		t.Fatalf("GetCredentials() error = %v", err)
	}
	if string(creds) != string(testCreds()) {
		t.Errorf("keychain credentials = %q, want %q", creds, testCreds())
	}
}

func TestGet(t *testing.T) {
	store, _, _ := newTestStore(t)

	if err := store.Create("personal", testCreds(), testSettings(), CreateOptions{Email: "bob@example.com"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	profile, err := store.Get("personal")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if profile.Name != "personal" {
		t.Errorf("Name = %q, want %q", profile.Name, "personal")
	}
	if profile.Email != "bob@example.com" {
		t.Errorf("Email = %q, want %q", profile.Email, "bob@example.com")
	}
	if profile.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if profile.IsActive {
		t.Error("IsActive = true, want false (no symlink)")
	}
}

func TestGet_NotFound(t *testing.T) {
	store, _, _ := newTestStore(t)

	_, err := store.Get("nonexistent")
	if err == nil {
		t.Fatal("Get() expected error for nonexistent profile, got nil")
	}
}

func TestList(t *testing.T) {
	store, _, _ := newTestStore(t)

	if err := store.Create("work", testCreds(), testSettings(), CreateOptions{Email: "work@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create("personal", testCreds(), testSettings(), CreateOptions{Email: "personal@example.com"}); err != nil {
		t.Fatal(err)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(profiles) != 2 {
		t.Fatalf("List() returned %d profiles, want 2", len(profiles))
	}

	// Profiles should be sorted by directory order (alphabetical on most FS).
	names := make(map[string]bool)
	for _, p := range profiles {
		names[p.Name] = true
	}
	if !names["work"] || !names["personal"] {
		t.Errorf("List() missing expected profiles, got %v", names)
	}
}

func TestList_ActiveMarked(t *testing.T) {
	keyringlib.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	claudeLink := filepath.Join(tmpDir, ".claude")
	ch := claude.NewClaudeHome(claudeLink)
	store := NewStore(profilesDir, ch)

	if err := store.Create("work", testCreds(), testSettings(), CreateOptions{Email: "work@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create("personal", testCreds(), testSettings(), CreateOptions{Email: "personal@example.com"}); err != nil {
		t.Fatal(err)
	}

	// Simulate activation: symlink claudeHome -> profiles/work
	workDir := filepath.Join(profilesDir, "work")
	if err := os.Symlink(workDir, claudeLink); err != nil {
		t.Fatal(err)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	for _, p := range profiles {
		if p.Name == "work" && !p.IsActive {
			t.Error("work profile should be active")
		}
		if p.Name == "personal" && p.IsActive {
			t.Error("personal profile should not be active")
		}
	}
}

func TestDelete(t *testing.T) {
	store, profilesDir, _ := newTestStore(t)

	if err := store.Create("doomed", testCreds(), testSettings(), CreateOptions{Email: "doomed@example.com"}); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete("doomed"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Directory should be gone.
	if _, err := os.Stat(filepath.Join(profilesDir, "doomed")); !os.IsNotExist(err) {
		t.Error("profile directory still exists after delete")
	}

	// Keychain should be cleared.
	if keychain.HasCredentials("doomed") {
		t.Error("keychain still has credentials after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	store, _, _ := newTestStore(t)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error for nonexistent profile, got nil")
	}
}

func TestRename(t *testing.T) {
	store, profilesDir, _ := newTestStore(t)

	if err := store.Create("old-name", testCreds(), testSettings(), CreateOptions{Email: "user@example.com"}); err != nil {
		t.Fatal(err)
	}

	if err := store.Rename("old-name", "new-name"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	// Old name should be gone.
	if _, err := os.Stat(filepath.Join(profilesDir, "old-name")); !os.IsNotExist(err) {
		t.Error("old profile directory still exists after rename")
	}
	if keychain.HasCredentials("old-name") {
		t.Error("keychain still has credentials under old name")
	}

	// New name should exist.
	if _, err := os.Stat(filepath.Join(profilesDir, "new-name")); os.IsNotExist(err) {
		t.Error("new profile directory does not exist after rename")
	}
	if !keychain.HasCredentials("new-name") {
		t.Error("keychain missing credentials under new name")
	}

	// Credentials should be the same.
	creds, err := keychain.GetCredentials("new-name")
	if err != nil {
		t.Fatalf("GetCredentials() error = %v", err)
	}
	if string(creds) != string(testCreds()) {
		t.Errorf("credentials changed after rename")
	}
}

func TestRename_OldNotFound(t *testing.T) {
	store, _, _ := newTestStore(t)

	err := store.Rename("nonexistent", "new-name")
	if err == nil {
		t.Fatal("Rename() expected error for nonexistent old name, got nil")
	}
}

func TestRename_NewAlreadyExists(t *testing.T) {
	store, _, _ := newTestStore(t)

	if err := store.Create("first", testCreds(), testSettings(), CreateOptions{Email: "a@example.com"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Create("second", testCreds(), testSettings(), CreateOptions{Email: "b@example.com"}); err != nil {
		t.Fatal(err)
	}

	err := store.Rename("first", "second")
	if err == nil {
		t.Fatal("Rename() expected error when new name already exists, got nil")
	}
}

func TestExists(t *testing.T) {
	store, _, _ := newTestStore(t)

	if store.Exists("nope") {
		t.Error("Exists() = true for missing profile, want false")
	}

	if err := store.Create("yep", testCreds(), testSettings(), CreateOptions{Email: "yep@example.com"}); err != nil {
		t.Fatal(err)
	}

	if !store.Exists("yep") {
		t.Error("Exists() = false for existing profile, want true")
	}
}

func TestActiveProfileName_Symlink(t *testing.T) {
	keyringlib.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	claudeLink := filepath.Join(tmpDir, ".claude")
	ch := claude.NewClaudeHome(claudeLink)
	store := NewStore(profilesDir, ch)

	if err := store.Create("active-profile", testCreds(), testSettings(), CreateOptions{Email: "active@example.com"}); err != nil {
		t.Fatal(err)
	}

	// Create symlink: .claude -> profiles/active-profile
	target := filepath.Join(profilesDir, "active-profile")
	if err := os.Symlink(target, claudeLink); err != nil {
		t.Fatal(err)
	}

	name, err := store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if name != "active-profile" {
		t.Errorf("ActiveProfileName() = %q, want %q", name, "active-profile")
	}
}

func TestActiveProfileName_NotSymlink(t *testing.T) {
	keyringlib.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a real directory (not a symlink) for claudeHome.
	claudeDir := filepath.Join(tmpDir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ch := claude.NewClaudeHome(claudeDir)
	store := NewStore(profilesDir, ch)

	name, err := store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if name != "" {
		t.Errorf("ActiveProfileName() = %q, want empty string", name)
	}
}

func TestActiveProfileName_NoClaudeHome(t *testing.T) {
	keyringlib.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// ClaudeHome doesn't exist at all.
	ch := claude.NewClaudeHome(filepath.Join(tmpDir, ".claude-nonexistent"))
	store := NewStore(profilesDir, ch)

	name, err := store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if name != "" {
		t.Errorf("ActiveProfileName() = %q, want empty string", name)
	}
}

func TestProfileDir(t *testing.T) {
	store, _, _ := newTestStore(t)

	got := store.ProfileDir("myprofile")
	want := filepath.Join(store.profilesDir, "myprofile")
	if got != want {
		t.Errorf("ProfileDir() = %q, want %q", got, want)
	}
}

func TestList_EmptyDir(t *testing.T) {
	store, _, _ := newTestStore(t)

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("List() returned %d profiles, want 0", len(profiles))
	}
}

func TestList_SkipsNonProfileDirs(t *testing.T) {
	store, profilesDir, _ := newTestStore(t)

	// Create a directory without metadata.json — should be skipped.
	if err := os.MkdirAll(filepath.Join(profilesDir, "not-a-profile"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a real profile.
	if err := store.Create("real", testCreds(), testSettings(), CreateOptions{Email: "real@example.com"}); err != nil {
		t.Fatal(err)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("List() returned %d profiles, want 1", len(profiles))
	}
	if profiles[0].Name != "real" {
		t.Errorf("List()[0].Name = %q, want %q", profiles[0].Name, "real")
	}
}

func TestCreate_WithOrgFields(t *testing.T) {
	keyringlib.MockInit()
	store, _, _ := newTestStore(t)

	creds := testCreds()
	settings := testSettings()
	opts := CreateOptions{
		Email:            "natik@respawn.io",
		OrgName:          "Lambda Inc",
		SubscriptionType: "enterprise",
	}

	if err := store.Create("work", creds, settings, opts); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p, err := store.Get("work")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if p.OrgName != "Lambda Inc" {
		t.Errorf("OrgName = %q, want %q", p.OrgName, "Lambda Inc")
	}
	if p.SubscriptionType != "enterprise" {
		t.Errorf("SubscriptionType = %q, want %q", p.SubscriptionType, "enterprise")
	}
	if p.Email != "natik@respawn.io" {
		t.Errorf("Email = %q, want %q", p.Email, "natik@respawn.io")
	}
}
