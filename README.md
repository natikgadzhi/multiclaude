# multiclaude

Switch between Claude Code accounts with a single command. Cross-platform, ergonomic, and secure.

## Problem

Claude Code stores credentials and config in `~/.claude/`. If you have multiple accounts (personal + work, multiple orgs), switching requires manually swapping config files, keychain entries, and settings. It's error-prone and tedious.

## What multiclaude does

multiclaude manages isolated **profiles** — each with its own Claude credentials, settings, and org context — and switches between them instantly by swapping symlinks and keychain entries.

```bash
# Add your current Claude account as a profile
multiclaude add work

# Switch to a different profile
multiclaude use personal

# See all profiles
multiclaude list

# Check which profile is active
multiclaude current
```

## Installation

```bash
brew install natikgadzhi/taps/multiclaude
```

Or with Go:
```bash
go install github.com/natikgadzhi/multiclaude@latest
```

## Quick Start

```bash
# 1. Save your current Claude session as a profile
multiclaude add work

# 2. Log into your personal account in Claude Code, then save it
multiclaude add personal

# 3. Switch between them any time
multiclaude use work
multiclaude use personal
```

## Commands

### `multiclaude add <name>`

Save the current Claude Code session as a named profile. Captures:
- OAuth credentials (stored in OS keychain, never on disk)
- User config (`~/.claude/settings.json`, `CLAUDE.md`)
- Organization/workspace context

```bash
multiclaude add work
multiclaude add personal --set-default
```

### `multiclaude use <name>`

Switch to a profile. Swaps credentials and config atomically via symlinks.

```bash
multiclaude use work
```

Shared settings (global CLAUDE.md, projects) are synced across profiles. Account-specific data (tokens, org caches) stays isolated.

### `multiclaude list`

Show all profiles with the active one highlighted.

```bash
multiclaude list
```
```
╭──────────┬────────────────────────┬─────────────────╮
│ PROFILE  │ EMAIL                  │ STATUS          │
├──────────┼────────────────────────┼─────────────────┤
│ work     │ natik@lambda.com       │ active          │
│ personal │ natik@natikgadzhi.com  │                 │
╰──────────┴────────────────────────┴─────────────────╯
```

### `multiclaude current`

Show the active profile name and associated email.

### `multiclaude remove <name>`

Delete a profile and its keychain entries. Cannot remove the active profile — switch first.

### `multiclaude rename <old> <new>`

Rename a profile.

### `multiclaude backup`

Create a snapshot of all profiles and config. Useful before upgrades.

```bash
multiclaude backup create "before-upgrade"
multiclaude backup list
multiclaude backup restore "before-upgrade"
```

### `multiclaude doctor`

Diagnose common issues: missing credentials, broken symlinks, stale profiles.

## Configuration

Stored at `~/.config/multiclaude/config.toml`:

```toml
# Default profile to use when none is specified
default_profile = "work"

# Claude home directory (rarely needs changing)
claude_home = "~/.claude"

# Auto-backup before every profile switch
auto_backup = true
```

## How it works

```
~/.config/multiclaude/
├── config.toml
└── profiles/
    ├── work/
    │   ├── credentials.json    # non-sensitive metadata (email, org)
    │   └── settings.json       # claude settings snapshot
    └── personal/
        ├── credentials.json
        └── settings.json

~/.claude/          → symlink to active profile's Claude state
Keychain:
  multiclaude/work/oauth    → OAuth token for work profile
  multiclaude/personal/oauth → OAuth token for personal profile
```

Credentials (OAuth tokens) live exclusively in the OS keychain via `go-keyring` (macOS Keychain, Linux Secret Service, Windows Credential Manager). They are never written to disk.

## Differences from claudini

| Feature | multiclaude | claudini |
|---------|-------------|----------|
| Language | Go | Rust |
| Install | `brew install` | curl script |
| Platforms | macOS, Linux, Windows | macOS only |
| Credentials | OS keychain (cross-platform) | macOS Keychain only |
| Config | TOML file | implicit |
| CLI UX | bordered tables, spinners, actionable errors | basic output |
| Shared library | cli-kit (consistent with other tools) | standalone |
| Diagnostics | `doctor` command | none |
| Auto-backup | on every switch (configurable) | manual only |
| Help | detailed examples, suggestions on error | minimal |

## Output

Supports `-o json` and `-o table` output formats. Default is auto-detected (table for TTY, JSON for pipes).

```bash
multiclaude list -o json | jq '.[].name'
```
