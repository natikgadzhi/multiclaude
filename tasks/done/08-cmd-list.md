# Task 08: `list` Command

## Objective
Implement `multiclaude list` — show all profiles in a bordered table.

## Acceptance Criteria
- Uses `cli-kit/table` for output
- Columns: Profile, Email, Status (shows "active" for current profile)
- JSON output: array of profile objects
- Empty state: "No profiles found. Run 'multiclaude add <name>' to create one."
- Uses `output.Resolve(cmd)` for format detection

## Dependencies
- Task 05 (profile model)
