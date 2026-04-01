# Task 10: `remove` Command

## Objective
Implement `multiclaude remove <name>` — delete a profile.

## Acceptance Criteria
- Deletes profile directory
- Deletes keychain entry
- Cannot remove active profile — error with "Switch to a different profile first"
- Confirmation prompt unless `--yes` flag is passed
- Prints "Removed profile: {name}"

## Dependencies
- Task 05 (profile model)
