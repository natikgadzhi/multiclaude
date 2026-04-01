package backup

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

// testCreds returns fake credential JSON bytes for testing.
func testCreds() []byte {
	return []byte(`{"claudeAiOauth":{"token":"test-token","email":"alice@example.com"}}`)
}

// testSettings returns a settings map for testing.
func testSettings() map[string]any {
	return map[string]any{
		"model": "opus",
	}
}

// newTestEnv creates a profile Store and backup Manager backed by temp dirs,
// with the keyring mock initialized. Returns the Manager, Store, backupsDir,
// and profilesDir.
func newTestEnv(t *testing.T) (*Manager, *profile.Store, string, string) {
	t.Helper()
	keyring.MockInit()

	tmpDir := t.TempDir()
	profilesDir := filepath.Join(tmpDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	backupsDir := filepath.Join(tmpDir, "backups")
	if err := os.MkdirAll(backupsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	claudeDir := filepath.Join(tmpDir, ".claude")
	ch := claude.NewClaudeHome(claudeDir)
	store := profile.NewStore(profilesDir, ch)
	mgr := NewManager(backupsDir, store)

	return mgr, store, backupsDir, profilesDir
}

func TestCreate(t *testing.T) {
	mgr, store, backupsDir, _ := newTestEnv(t)

	// Create two profiles.
	if err := store.Create("work", testCreds(), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.Create("personal", testCreds(), testSettings(), "personal@example.com"); err != nil {
		t.Fatal(err)
	}

	// Create backup.
	if err := mgr.Create("pre-upgrade"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify backup directory structure.
	backupDir := filepath.Join(backupsDir, "pre-upgrade")
	if _, err := os.Stat(filepath.Join(backupDir, "metadata.json")); os.IsNotExist(err) {
		t.Fatal("metadata.json not created")
	}
	if _, err := os.Stat(filepath.Join(backupDir, "profiles", "work")); os.IsNotExist(err) {
		t.Fatal("profiles/work not created in backup")
	}
	if _, err := os.Stat(filepath.Join(backupDir, "profiles", "personal")); os.IsNotExist(err) {
		t.Fatal("profiles/personal not created in backup")
	}
	if _, err := os.Stat(filepath.Join(backupDir, "keychain", "work.b64")); os.IsNotExist(err) {
		t.Fatal("keychain/work.b64 not created")
	}
	if _, err := os.Stat(filepath.Join(backupDir, "keychain", "personal.b64")); os.IsNotExist(err) {
		t.Fatal("keychain/personal.b64 not created")
	}

	// Verify metadata content.
	metaBytes, err := os.ReadFile(filepath.Join(backupDir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta backupMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Profiles != 2 {
		t.Errorf("metadata.Profiles = %d, want 2", meta.Profiles)
	}
	if meta.CreatedAt.IsZero() {
		t.Error("metadata.CreatedAt is zero")
	}

	// Verify profile files were copied.
	workMeta, err := os.ReadFile(filepath.Join(backupDir, "profiles", "work", "metadata.json"))
	if err != nil {
		t.Fatalf("reading backed-up work metadata: %v", err)
	}
	var wm struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(workMeta, &wm); err != nil {
		t.Fatal(err)
	}
	if wm.Email != "work@example.com" {
		t.Errorf("backed-up work email = %q, want %q", wm.Email, "work@example.com")
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	mgr, _, _, _ := newTestEnv(t)

	if err := mgr.Create("dup"); err != nil {
		t.Fatal(err)
	}

	err := mgr.Create("dup")
	if err == nil {
		t.Fatal("Create() expected error for duplicate backup name, got nil")
	}
}

func TestCreate_NoProfiles(t *testing.T) {
	mgr, _, backupsDir, _ := newTestEnv(t)

	if err := mgr.Create("empty"); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	metaBytes, err := os.ReadFile(filepath.Join(backupsDir, "empty", "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta backupMetadata
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Profiles != 0 {
		t.Errorf("metadata.Profiles = %d, want 0", meta.Profiles)
	}
}

func TestList(t *testing.T) {
	mgr, store, _, _ := newTestEnv(t)

	if err := store.Create("work", testCreds(), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Create("first"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("second"); err != nil {
		t.Fatal(err)
	}

	backups, err := mgr.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(backups) != 2 {
		t.Fatalf("List() returned %d backups, want 2", len(backups))
	}

	// Newest first.
	if backups[0].CreatedAt.Before(backups[1].CreatedAt) {
		t.Error("List() not sorted newest first")
	}

	names := map[string]bool{}
	for _, b := range backups {
		names[b.Name] = true
	}
	if !names["first"] || !names["second"] {
		t.Errorf("List() missing expected backups, got %v", names)
	}
}

func TestList_Empty(t *testing.T) {
	mgr, _, _, _ := newTestEnv(t)

	backups, err := mgr.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("List() returned %d backups, want 0", len(backups))
	}
}

func TestRestore(t *testing.T) {
	mgr, store, _, profilesDir := newTestEnv(t)

	// Create profiles and backup.
	if err := store.Create("work", testCreds(), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := store.Create("personal", testCreds(), testSettings(), "personal@example.com"); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Create("snapshot"); err != nil {
		t.Fatal(err)
	}

	// Delete all profiles to simulate loss.
	if err := store.Delete("work"); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete("personal"); err != nil {
		t.Fatal(err)
	}

	// Verify profiles are gone.
	profiles, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected 0 profiles after delete, got %d", len(profiles))
	}

	// Restore.
	if err := mgr.Restore("snapshot"); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	// Verify profiles are back.
	profiles, err = store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 2 {
		t.Fatalf("Restore() resulted in %d profiles, want 2", len(profiles))
	}

	// Verify profile data.
	for _, p := range profiles {
		metaBytes, err := os.ReadFile(filepath.Join(profilesDir, p.Name, "metadata.json"))
		if err != nil {
			t.Fatalf("reading restored metadata for %q: %v", p.Name, err)
		}
		var meta struct {
			Email string `json:"email"`
		}
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			t.Fatal(err)
		}
		if meta.Email == "" {
			t.Errorf("restored profile %q has empty email", p.Name)
		}
	}

	// Verify keychain credentials restored.
	if !keychain.HasCredentials("work") {
		t.Error("keychain missing credentials for 'work' after restore")
	}
	if !keychain.HasCredentials("personal") {
		t.Error("keychain missing credentials for 'personal' after restore")
	}

	creds, err := keychain.GetCredentials("work")
	if err != nil {
		t.Fatalf("GetCredentials('work') error = %v", err)
	}
	if string(creds) != string(testCreds()) {
		t.Errorf("restored credentials don't match original")
	}
}

func TestRestore_NotFound(t *testing.T) {
	mgr, _, _, _ := newTestEnv(t)

	err := mgr.Restore("nonexistent")
	if err == nil {
		t.Fatal("Restore() expected error for nonexistent backup, got nil")
	}
}

func TestRestore_ReplacesExisting(t *testing.T) {
	mgr, store, _, _ := newTestEnv(t)

	// Create one profile and back it up.
	if err := store.Create("work", testCreds(), testSettings(), "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.Create("snap"); err != nil {
		t.Fatal(err)
	}

	// Create a second profile (not in backup).
	if err := store.Create("extra", testCreds(), testSettings(), "extra@example.com"); err != nil {
		t.Fatal(err)
	}

	// Restore should remove "extra" and keep only "work".
	if err := mgr.Restore("snap"); err != nil {
		t.Fatal(err)
	}

	profiles, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 {
		t.Fatalf("Restore() resulted in %d profiles, want 1", len(profiles))
	}
	if profiles[0].Name != "work" {
		t.Errorf("restored profile name = %q, want %q", profiles[0].Name, "work")
	}
}

func TestDelete(t *testing.T) {
	mgr, _, backupsDir, _ := newTestEnv(t)

	if err := mgr.Create("to-delete"); err != nil {
		t.Fatal(err)
	}

	if err := mgr.Delete("to-delete"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(backupsDir, "to-delete")); !os.IsNotExist(err) {
		t.Error("backup directory still exists after delete")
	}
}

func TestDelete_NotFound(t *testing.T) {
	mgr, _, _, _ := newTestEnv(t)

	err := mgr.Delete("nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error for nonexistent backup, got nil")
	}
}
