# Task 17: Error Messages Polish

## Objective
Ensure every error path has actionable, human-readable guidance.

## Acceptance Criteria
- Every command error includes: what happened, why, what to do
- "Profile not found" → lists available profiles
- "No active session" → "Log into Claude Code first, then run multiclaude add <name>"
- "Cannot remove active profile" → "Switch to another profile first: multiclaude use <name>"
- "Profile already exists" → "Use a different name, or remove it first: multiclaude remove <name>"
- Config errors → show minimal config example
- Keychain errors → platform-specific guidance

## Dependencies
- All command tasks
