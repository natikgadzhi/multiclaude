# multiclaude

Switch between Claude Code accounts with a single command. Cross-platform, ergonomic, and secure.

## Problem

Claude Code stores credentials and config in `~/.claude/`. If you have multiple accounts (personal + work, multiple orgs), switching requires manually swapping config files, keychain entries, and settings. It's error-prone and tedious.

## What multiclaude does

multiclaude manages isolated **profiles** -- each with its own Claude credentials, settings, and org context -- and switches between them instantly by copying credentials and settings in and out of `~/.claude/`.

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
- User settings (`~/.claude/settings.json`)
- Account email (from the credentials)

```bash
multiclaude add work
multiclaude add personal --set-default
```

### `multiclaude use <name>`

Switch to a profile. Saves the current profile's state, then restores the target profile's credentials and settings into `~/.claude/`.

```bash
multiclaude use work
```

### `multiclaude list`

Show all profiles with the active one highlighted.

```bash
multiclaude list
```
```
PROFILE    EMAIL                   STATUS
work       natik@lambda.com        active
personal   natik@natikgadzhi.com
```

### `multiclaude current`

Show the active profile name and associated email.

```bash
multiclaude current
```

### `multiclaude remove <name>`

Delete a profile and its keychain entries. Cannot remove the active profile -- switch first.

```bash
multiclaude remove old-account
```

### `multiclaude rename <old> <new>`

Rename a profile. Updates the profile directory, keychain entry, and active tracking.

```bash
multiclaude rename work work-main
```

### `multiclaude backup`

Create and manage snapshots of all profiles and their credentials. Useful before upgrades.

```bash
multiclaude backup create before-upgrade
multiclaude backup list
multiclaude backup restore before-upgrade
multiclaude backup delete before-upgrade
```

### `multiclaude doctor`

Diagnose common issues: missing credentials, broken state, stale profiles.

```bash
multiclaude doctor
```

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

multiclaude stores profile data under `~/.config/multiclaude/` and credentials in the OS keychain. When you switch profiles, it copies credentials and settings into `~/.claude/`.

```
~/.config/multiclaude/
├── config.toml                 # multiclaude settings
├── active                      # name of the currently active profile
├── profiles/
│   ├── work/
│   │   ├── metadata.json       # email, creation time
│   │   └── settings.json       # Claude settings snapshot
│   └── personal/
│       ├── metadata.json
│       └── settings.json
└── backups/
    └── before-upgrade/
        ├── metadata.json
        ├── profiles/           # copy of all profile directories
        └── keychain/           # exported keychain credentials

~/.claude/                      # Claude Code reads from here
├── .credentials.json           # written by multiclaude on switch
├── settings.json               # written by multiclaude on switch
├── CLAUDE.md
└── projects/

OS Keychain:
  multiclaude/work/oauth        # OAuth token for work profile
  multiclaude/personal/oauth    # OAuth token for personal profile
```

**Switching flow:**
1. `multiclaude use <target>` saves the current profile's credentials (to keychain) and settings (to its profile directory)
2. Retrieves the target profile's credentials from the keychain
3. Writes the target profile's credentials and settings into `~/.claude/`
4. Updates `~/.config/multiclaude/active` to track the active profile

Credentials (OAuth tokens) live exclusively in the OS keychain via `go-keyring` (macOS Keychain, Linux Secret Service, Windows Credential Manager). They are never written to disk in plaintext.

## Output

Supports `-o json` and `-o table` output formats. Default is auto-detected (table for TTY, JSON for pipes).

```bash
multiclaude list -o json | jq '.[].name'
```

## Version

```bash
multiclaude version
multiclaude --version
```
