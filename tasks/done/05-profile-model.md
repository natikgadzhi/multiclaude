# Task 05: Profile Data Model and Storage

## Objective
Define the profile model and filesystem operations for creating, reading, listing, and deleting profiles.

## Acceptance Criteria
- `internal/profile/profile.go` with:
  ```go
  type Profile struct {
      Name      string
      Email     string    // extracted from credentials
      CreatedAt time.Time
      IsActive  bool      // true if this is the current profile
  }

  type Store struct {
      ProfilesDir string
      ClaudeHome  *claude.ClaudeHome
  }
  ```
- `NewStore(profilesDir string, claudeHome *claude.ClaudeHome) *Store`
- `(s *Store) Create(name string, creds []byte, settings map[string]any) error`
  - Creates `profiles/{name}/` directory
  - Writes `credentials.json` (non-sensitive metadata only — email, org)
  - Writes `settings.json` snapshot
  - Stores actual OAuth token in keychain via keychain package
- `(s *Store) Get(name string) (*Profile, error)`
- `(s *Store) List() ([]Profile, error)` — reads all profile dirs, marks active
- `(s *Store) Delete(name string) error` — removes dir + keychain entry
- `(s *Store) Rename(old, new string) error` — renames dir + keychain key
- `(s *Store) Exists(name string) bool`
- `(s *Store) ActiveProfile() (string, error)` — determine which profile is active by checking symlink target
- Tests with t.TempDir() and keyring.MockInit()

## Dependencies
- Task 03 (claude integration)
- Task 04 (keychain)
