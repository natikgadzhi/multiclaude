# Task 13: `backup` Commands

## Objective
Implement `multiclaude backup create/list/restore` for profile snapshots.

## Acceptance Criteria
- `backup create <name>` — snapshot all profiles + config to `~/.config/multiclaude/backups/{name}/`
  - Copies all profile directories
  - Exports keychain credentials to encrypted backup (or just re-read from keychain on restore)
  - Records timestamp
- `backup list` — show available backups in a table (Name, Created, Profile Count)
- `backup restore <name>` — restore from backup
  - Overwrites current profiles
  - Restores keychain entries
  - Warning: this replaces all current profiles
- `backup delete <name>` — remove a backup

## Dependencies
- Task 05 (profile model)
- Task 04 (keychain)
