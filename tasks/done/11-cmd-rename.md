# Task 11: `rename` Command

## Objective
Implement `multiclaude rename <old> <new>` — rename a profile.

## Acceptance Criteria
- Renames profile directory
- Updates keychain key (delete old, store under new name)
- If renaming the active profile, update symlink target
- Validates new name doesn't already exist
- Prints "Renamed profile: {old} → {new}"

## Dependencies
- Task 05 (profile model)
