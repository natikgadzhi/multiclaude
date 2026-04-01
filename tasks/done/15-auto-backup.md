# Task 15: Auto-Backup on Switch

## Objective
Automatically create a backup before every profile switch (configurable).

## Acceptance Criteria
- When `auto_backup = true` in config.toml (default), `use` command creates a timestamped backup before switching
- Backup name: `auto-{timestamp}` (e.g., `auto-2026-04-01T15:04:05`)
- Keep only the last 5 auto-backups, prune older ones
- Skipped when `auto_backup = false`
- Debug log: "Auto-backup created: auto-2026-04-01T15:04:05"

## Dependencies
- Task 07 (use command)
- Task 13 (backup commands)
