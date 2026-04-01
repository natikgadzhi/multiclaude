# Task 12: Settings Sync Engine

## Objective
Manage which settings are shared across profiles vs isolated per account.

## Acceptance Criteria
- `internal/sync/sync.go` with:
  ```go
  // SharedFields returns the list of settings.json keys that sync across profiles.
  func SharedFields() []string

  // AccountFields returns the keys that are account-specific.
  func AccountFields() []string

  // MergeSettings merges shared fields from src into dst, preserving dst's account-specific fields.
  func MergeSettings(src, dst map[string]any) map[string]any
  ```
- Shared fields (sync across profiles): user preferences (theme, editor mode), project configs, CLAUDE.md content
- Account-specific fields (isolated): OAuth tokens, user IDs, organization info, API endpoints
- The sync happens during `use` command: outgoing profile's shared settings are merged into incoming profile
- Tests: merge preserves account fields, copies shared fields, handles missing fields

## Notes
- Read Claude Code's actual settings.json to determine which fields are which
- When in doubt, mark a field as account-specific (safer)

## Dependencies
- Task 03 (claude integration)
