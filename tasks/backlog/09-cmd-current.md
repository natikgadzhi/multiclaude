# Task 09: `current` Command

## Objective
Implement `multiclaude current` — show the active profile.

## Acceptance Criteria
- Prints active profile name and email
- Table mode: single-row table with Name, Email
- JSON mode: `{"name": "work", "email": "natik@lambda.com"}`
- If no profile is active: "No active profile. Run 'multiclaude add <name>' to create one."
- Exit code 0 always (informational command)

## Dependencies
- Task 05 (profile model)
