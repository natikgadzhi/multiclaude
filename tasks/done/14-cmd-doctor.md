# Task 14: `doctor` Command

## Objective
Implement `multiclaude doctor` — diagnose common issues.

## Acceptance Criteria
- Checks:
  - `~/.claude` exists and is readable
  - Config file is valid (or missing with defaults)
  - Each profile directory is intact
  - Each profile has matching keychain entry
  - Active profile symlink points to a valid profile
  - Claude Code CLI is available in PATH
- Output: table of checks with pass/fail/warning status
- For failures: print actionable fix suggestions
- Exit code 0 if all pass, 1 if any fail

## Dependencies
- Task 05 (profile model)
- Task 04 (keychain)
