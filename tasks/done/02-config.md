# Task 02: Config Package

## Objective
Load multiclaude's own config from `~/.config/multiclaude/config.toml`.

## Acceptance Criteria
- `internal/config/config.go` with types:
  ```go
  type Config struct {
      DefaultProfile string `toml:"default_profile"`
      ClaudeHome     string `toml:"claude_home"`     // default: ~/.claude
      AutoBackup     bool   `toml:"auto_backup"`     // default: true
  }
  ```
- `Load() (*Config, error)` — uses `cli-kit/config.Load()`, applies defaults
- `ProfilesDir() string` — returns `~/.config/multiclaude/profiles`
- `BackupsDir() string` — returns `~/.config/multiclaude/backups`
- Creates directories if they don't exist
- Config file is optional — missing file uses all defaults
- Tests with t.TempDir()

## Dependencies
- Task 01 (bootstrap)
