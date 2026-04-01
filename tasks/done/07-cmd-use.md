# Task 07: `use` Command

## Objective
Implement `multiclaude use <name>` — switch active profile.

## Acceptance Criteria
- Validates profile exists
- If current profile is the same, print "already active" and return
- Save current state (credentials + settings) to current profile before switching (auto-save)
- Swap keychain: restore target profile's OAuth credentials to Claude's keychain location
- Swap symlink: point `~/.claude` to the target profile's state (or restore settings to ~/.claude)
- Show spinner during switch
- Print success: "Switched to profile: {name} ({email})"
- Error if profile doesn't exist (list available profiles in error message)

## Design Decision: Symlink vs Copy
Two approaches:
1. **Symlink `~/.claude/`** to a profile directory — simpler but Claude Code might not like it
2. **Copy files** in/out of `~/.claude/` on switch — more compatible

Read Claude Code's actual behavior to decide. If `~/.claude/` must be a real directory (not symlink), use the copy approach with atomic operations.

## Dependencies
- Task 05 (profile model)
- Task 04 (keychain)
