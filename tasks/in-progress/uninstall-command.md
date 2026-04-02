# Task: `multiclaude uninstall` Command

## Objective

Add a command that cleanly removes multiclaude from the system, restoring the
user's Claude Code to a vanilla (unmanaged) state.

## Background

multiclaude's footprint is:
- `~/.config/multiclaude/` — profile metadata and active state file
- OS keychain entries under `multiclaude/<name>/oauth`
- `~/.claude/.credentials.json` — written on switch, but originally from Claude Code

`~/.claude/` is never moved or symlinked — it remains a plain directory. On uninstall
the current active profile's credentials are already there, so no restore step is needed
as long as the active profile is set.

## Acceptance Criteria

- `multiclaude uninstall` refuses if more than one profile exists:
  ```
  Error: 2 profiles exist. Remove extras first with `multiclaude remove <name>`,
  then re-run uninstall.
  ```
- If zero or one profile exists and it is active, the command:
  1. Deletes all keychain entries for remaining profiles
  2. Deletes `~/.config/multiclaude/` entirely
  3. Prints a confirmation message
- If one profile exists but is NOT active (credentials not in `~/.claude/`):
  - Warn the user and ask for confirmation before proceeding, since `~/.claude/`
    may not have valid credentials
- Dry-run flag `--dry-run` prints what would be deleted without doing it
- No changes to `~/.claude/` — only multiclaude's own state is removed

## What to add

- `cmd/uninstall.go` — new command
- Register in `cmd/root.go`
- Tests in `cmd/` or `cmd/integration_test.go`
