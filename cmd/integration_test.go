package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/keychain"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/zalando/go-keyring"
)

// testCreds returns fake credential JSON bytes matching Claude Code's format.
func testCreds(email string) []byte {
	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"token":        "test-token-" + email,
			"expiresAt":    "2099-01-01T00:00:00Z",
			"refreshToken": "test-refresh-" + email,
			"email":        email,
		},
	}
	b, _ := json.MarshalIndent(creds, "", "  ")
	return b
}

// testSettings returns a settings map for testing.
func testSettings() map[string]any {
	return map[string]any{
		"model": "opus",
	}
}

// testEnv sets up an isolated test environment with temp dirs, mock keyring,
// and returns a profile Store, the ClaudeHome, and key directory paths.
type testEnv struct {
	store      *profile.Store
	claudeHome *claude.ClaudeHome
	profileDir string
	activeFile string
	claudeDir  string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	keyring.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	claudeDir := filepath.Join(tmpDir, ".claude")
	activeFile := filepath.Join(tmpDir, "active")

	for _, d := range []string{profilesDir, claudeDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Write fake credentials and settings into the claude home so add-like
	// operations can read them.
	credsData := testCreds("setup@example.com")
	if err := os.WriteFile(filepath.Join(claudeDir, ".credentials.json"), credsData, 0o600); err != nil {
		t.Fatal(err)
	}
	settingsData, _ := json.MarshalIndent(testSettings(), "", "  ")
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), settingsData, 0o644); err != nil {
		t.Fatal(err)
	}

	ch := claude.NewClaudeHome(claudeDir)
	store := profile.NewStore(profilesDir, ch)
	store.SetActiveFile(activeFile)

	return &testEnv{
		store:      store,
		claudeHome: ch,
		profileDir: profilesDir,
		activeFile: activeFile,
		claudeDir:  claudeDir,
	}
}

// TestIntegration_AddAndList tests that profiles created via Store.Create
// are visible via Store.List with correct metadata.
func TestIntegration_AddAndList(t *testing.T) {
	env := newTestEnv(t)

	// Add a profile.
	err := env.store.Create("work", testCreds("work@example.com"), testSettings(), "work@example.com")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// List should show one profile.
	profiles, err := env.store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 1 {
		t.Fatalf("List() returned %d profiles, want 1", len(profiles))
	}
	if profiles[0].Name != "work" {
		t.Errorf("profile name = %q, want %q", profiles[0].Name, "work")
	}
	if profiles[0].Email != "work@example.com" {
		t.Errorf("profile email = %q, want %q", profiles[0].Email, "work@example.com")
	}

	// Keychain should have credentials.
	if !keychain.HasCredentials("work") {
		t.Error("keychain missing credentials for 'work'")
	}
}

// TestIntegration_AddUseCurrent tests switching between two profiles
// and verifying ActiveProfileName changes accordingly.
func TestIntegration_AddUseCurrent(t *testing.T) {
	env := newTestEnv(t)

	// Add two profiles.
	if err := env.store.Create("work", testCreds("work@example.com"), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := env.store.Create("personal", testCreds("personal@example.com"), testSettings(), "personal@example.com"); err != nil {
		t.Fatal(err)
	}

	// Mark "work" as active.
	if err := env.store.WriteActive("work"); err != nil {
		t.Fatalf("WriteActive() error = %v", err)
	}

	active, err := env.store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if active != "work" {
		t.Errorf("active = %q, want %q", active, "work")
	}

	// Simulate switching: save state for current, restore target, write active.
	// Save current state (work).
	if err := env.store.SaveState("work"); err != nil {
		t.Logf("SaveState warning: %v", err)
	}

	// Restore personal's state.
	if err := env.store.RestoreState("personal"); err != nil {
		t.Fatalf("RestoreState('personal') error = %v", err)
	}

	// Update active to personal.
	if err := env.store.WriteActive("personal"); err != nil {
		t.Fatalf("WriteActive('personal') error = %v", err)
	}

	active, err = env.store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if active != "personal" {
		t.Errorf("active = %q after switch, want %q", active, "personal")
	}

	// Verify credentials in claude home match personal's credentials.
	creds, err := env.claudeHome.ReadCredentials()
	if err != nil {
		t.Fatalf("ReadCredentials() error = %v", err)
	}
	if string(creds) != string(testCreds("personal@example.com")) {
		t.Error("claude home credentials do not match personal profile after switch")
	}
}

// TestIntegration_AddAndRemove tests that a profile can be added then removed,
// resulting in an empty list and cleared keychain.
func TestIntegration_AddAndRemove(t *testing.T) {
	env := newTestEnv(t)

	// Add a profile.
	if err := env.store.Create("temp", testCreds("temp@example.com"), testSettings(), "temp@example.com"); err != nil {
		t.Fatal(err)
	}

	// Verify it exists.
	if !env.store.Exists("temp") {
		t.Fatal("profile 'temp' should exist after create")
	}

	// Delete it.
	if err := env.store.Delete("temp"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify list is empty.
	profiles, err := env.store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("List() returned %d profiles after remove, want 0", len(profiles))
	}

	// Verify keychain is cleared.
	if keychain.HasCredentials("temp") {
		t.Error("keychain still has credentials for 'temp' after remove")
	}
}

// TestIntegration_AddAndRename tests that renaming a profile removes the old
// name and creates the new one with the same data and credentials.
func TestIntegration_AddAndRename(t *testing.T) {
	env := newTestEnv(t)

	// Add a profile.
	if err := env.store.Create("old-name", testCreds("user@example.com"), testSettings(), "user@example.com"); err != nil {
		t.Fatal(err)
	}

	// Rename it.
	if err := env.store.Rename("old-name", "new-name"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	// Old name should be gone.
	if env.store.Exists("old-name") {
		t.Error("old profile name still exists after rename")
	}
	if keychain.HasCredentials("old-name") {
		t.Error("keychain still has credentials under old name")
	}

	// New name should exist with the same data.
	if !env.store.Exists("new-name") {
		t.Error("new profile name does not exist after rename")
	}

	p, err := env.store.Get("new-name")
	if err != nil {
		t.Fatalf("Get('new-name') error = %v", err)
	}
	if p.Email != "user@example.com" {
		t.Errorf("renamed profile email = %q, want %q", p.Email, "user@example.com")
	}

	creds, err := keychain.GetCredentials("new-name")
	if err != nil {
		t.Fatalf("GetCredentials('new-name') error = %v", err)
	}
	if string(creds) != string(testCreds("user@example.com")) {
		t.Error("credentials changed after rename")
	}
}

// TestIntegration_DoctorHealthy tests that a well-formed setup with valid
// profiles and credentials passes all checks in the doctor logic.
// Rather than calling the cobra command, we replicate the key check functions
// to verify the setup is healthy.
func TestIntegration_DoctorHealthy(t *testing.T) {
	env := newTestEnv(t)

	// Create two valid profiles.
	if err := env.store.Create("work", testCreds("work@example.com"), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := env.store.Create("personal", testCreds("personal@example.com"), testSettings(), "personal@example.com"); err != nil {
		t.Fatal(err)
	}

	// Mark one as active.
	if err := env.store.WriteActive("work"); err != nil {
		t.Fatal(err)
	}

	// Check 1: Claude home exists.
	if !env.claudeHome.Exists() {
		t.Error("doctor: claude home does not exist")
	}

	// Check 2: Profiles directory exists.
	if _, err := os.Stat(env.profileDir); os.IsNotExist(err) {
		t.Error("doctor: profiles directory does not exist")
	}

	// Check 3: Each profile has a directory and keychain entry.
	profiles, err := env.store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(profiles))
	}

	for _, p := range profiles {
		dir := env.store.ProfileDir(p.Name)
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			t.Errorf("doctor: profile %q directory missing or not a directory", p.Name)
		}

		if !keychain.HasCredentials(p.Name) {
			t.Errorf("doctor: profile %q keychain entry missing", p.Name)
		}
	}

	// Check 4: Active profile is consistent.
	activeName, err := env.store.ActiveProfileName()
	if err != nil {
		t.Fatalf("ActiveProfileName() error = %v", err)
	}
	if activeName != "work" {
		t.Errorf("active profile = %q, want %q", activeName, "work")
	}
	if !env.store.Exists(activeName) {
		t.Error("active profile does not exist in store")
	}
}

