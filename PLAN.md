# multiclaude — Implementation Plan

## Overview

A Go CLI tool for switching between Claude Code accounts via named profiles. Uses cli-kit for all shared infrastructure.

## Phases

### Phase 0: Bootstrap
- **Task 01**: Project scaffolding — main.go, Cobra root command, Makefile, .goreleaser.yml, CI workflow
- **Task 02**: Config package — load `~/.config/multiclaude/config.toml`, types, defaults

### Phase 1: Core — Understanding Claude Code
- **Task 03**: Claude Code integration — detect and read `~/.claude/` structure, parse `.credentials.json`, read `settings.json`
- **Task 04**: Keychain operations — store/retrieve/delete OAuth credentials via cli-kit/auth, keyed per profile

### Phase 2: Profile Management
- **Task 05**: Profile data model and storage — `~/.config/multiclaude/profiles/{name}/` structure, create/read/list/delete
- **Task 06**: `add` command — capture current Claude session (credentials + settings) into a named profile
- **Task 07**: `use` command — switch active profile (swap symlinks, swap keychain, sync shared settings)
- **Task 08**: `list` command — show all profiles in a bordered table with active indicator
- **Task 09**: `current` command — show active profile name and email
- **Task 10**: `remove` command — delete a profile and its keychain entries
- **Task 11**: `rename` command — rename a profile (directory + keychain keys)

### Phase 3: Settings Sync
- **Task 12**: Sync engine — identify shared vs account-specific fields, merge on switch
  - Shared: global CLAUDE.md, project configs, user preferences
  - Account-specific: OAuth tokens, user IDs, org caches

### Phase 4: Backup & Diagnostics
- **Task 13**: `backup create/list/restore` commands — snapshot all profiles
- **Task 14**: `doctor` command — check for broken symlinks, missing credentials, stale profiles
- **Task 15**: Auto-backup on switch (configurable via `auto_backup` in config.toml)

### Phase 5: Polish
- **Task 16**: Integration tests — full round-trip: add profile, switch, verify, remove
- **Task 17**: Error messages — actionable guidance for every failure mode
- **Task 18**: README polish, help text, examples
- **Task 19**: Homebrew formula and release workflow

## Dependency Chain

```
01 (bootstrap)
├── 02 (config) ──────────────────────────────────┐
├── 03 (claude integration) ──┐                    │
├── 04 (keychain) ────────────┤                    │
│                             ├── 05 (profile model)
│                             │   ├── 06 (add)
│                             │   ├── 07 (use) ← 12 (sync)
│                             │   ├── 08 (list)
│                             │   ├── 09 (current)
│                             │   ├── 10 (remove)
│                             │   └── 11 (rename)
│                             │
│                             ├── 13 (backup)
│                             └── 14 (doctor)
│
15 (auto-backup) ← 07 + 13
16 (integration tests) ← all implementation tasks
17 (error messages) ← all commands
18 (README polish) ← 16
19 (homebrew + release) ← 18
```

## Parallelism Opportunities

- Tasks 02, 03, 04 can run in parallel after 01
- Tasks 06-11 can run in parallel after 05 (each command is independent)
- Tasks 13, 14 can run in parallel
- Task 12 (sync) can start after 03, runs in parallel with commands
