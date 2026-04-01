# Task 03: Claude Code Integration

## Objective
Read and understand Claude Code's file structure at `~/.claude/`.

## Acceptance Criteria
- `internal/claude/claude.go` with:
  ```go
  type ClaudeHome struct {
      Path string // e.g. ~/.claude
  }

  type Credentials struct {
      // OAuth token fields — read from .credentials.json
      // Exact structure TBD: read actual .credentials.json to determine
  }

  type Settings struct {
      // Read from settings.json — keep as map[string]any for flexibility
  }
  ```
- `NewClaudeHome(path string) *ClaudeHome`
- `(ch *ClaudeHome) ReadCredentials() (*Credentials, error)` — parse `.credentials.json`
- `(ch *ClaudeHome) ReadSettings() (map[string]any, error)` — parse `settings.json`
- `(ch *ClaudeHome) IsSymlink() bool` — check if ~/.claude is a symlink
- `(ch *ClaudeHome) SymlinkTarget() (string, error)` — where it points
- `(ch *ClaudeHome) ActiveEmail() (string, error)` — extract email/identity from credentials
- Tests: mock filesystem with t.TempDir(), create fake .credentials.json and settings.json

## Notes
- Read the ACTUAL `~/.claude/.credentials.json` on this machine to understand the real format
- Read `~/.claude/settings.json` to understand what's in there
- Do NOT read or store actual credentials in test fixtures — use fake values

## Dependencies
- Task 01 (bootstrap)
