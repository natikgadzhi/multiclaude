# Task 06: `add` Command

## Objective
Implement `multiclaude add <name>` — capture current Claude session as a named profile.

## Acceptance Criteria
- Reads current credentials from Claude home
- Reads current settings
- Creates profile via Store.Create()
- Stores OAuth token in keychain
- If `--set-default` flag is passed, updates config.toml default_profile
- If this is the first profile, set up the symlink and make it active
- Validates: name is alphanumeric+dashes, profile doesn't already exist, Claude session exists
- Shows success message with profile name and email
- Error if no active Claude session found (actionable: "Log into Claude Code first")

## Dependencies
- Task 05 (profile model)
- Task 02 (config)
