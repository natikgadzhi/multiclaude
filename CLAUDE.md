@README.md

# Project: multiclaude

A Go CLI tool for switching between Claude Code accounts via named profiles.

## Architecture

```
cmd/
  root.go          — Cobra root command, global flags (--config, -o, --debug)
  add.go           — `add` command: capture current Claude session as profile
  use.go           — `use` command: switch active profile
  list.go          — `list` command: show all profiles
  current.go       — `current` command: show active profile
  remove.go        — `remove` command: delete a profile
  rename.go        — `rename` command: rename a profile
  doctor.go        — `doctor` command: diagnose issues
  version.go       — version command (via cli-kit)
internal/
  profile/         — Profile CRUD: create, read, list, delete, rename
  keychain/        — Credential storage: store/retrieve OAuth tokens from OS keychain
  claude/          — Claude Code integration: read/write ~/.claude/, detect active session
  sync/            — Settings sync: shared vs account-specific field management
  config/          — TOML config loading for multiclaude's own config
```

## Dependencies

- `github.com/natikgadzhi/cli-kit` — Shared CLI standards (ALL applicable packages)
- `github.com/spf13/cobra` — CLI framework
- `github.com/BurntSushi/toml` — Config parsing (via cli-kit/config)

## cli-kit Usage (MANDATORY)

This project MUST use cli-kit for all covered features:
- `cli-kit/table` — all table output
- `cli-kit/output` — `-o/--output` flag, TTY detection
- `cli-kit/progress` — spinners during profile switch
- `cli-kit/errors` — all error handling, ExitWithError in root.go
- `cli-kit/debug` — `--debug` flag and Debug: logging
- `cli-kit/config` — TOML config loading, `--config` flag
- `cli-kit/auth` — OS keychain access (StoreToken, DeleteToken, MaskToken)
- `cli-kit/version` — version command and --version flag

Do NOT reimplement any of these locally.

## Conventions

- All tests use `testing` stdlib + table-driven tests
- Use `t.TempDir()` for filesystem tests
- Keychain tests use `keyring.MockInit()` for in-memory mock
- Profile operations are atomic: write temp file, then rename
- Credentials never touch disk — keychain only
- Copy-based switching: credentials and settings are written into `~/.claude/`
- Active profile tracked via `~/.config/multiclaude/active` state file
- Config at `~/.config/multiclaude/config.toml`

## Claude Code File Layout

multiclaude needs to understand Claude Code's file structure:
```
~/.claude/
├── .credentials.json    — OAuth tokens (THIS is what goes to keychain)
├── settings.json        — User preferences
├── CLAUDE.md            — Global instructions
└── projects/            — Project-specific state
```

The `.credentials.json` contains the OAuth bearer token and refresh token.
Settings and CLAUDE.md are user preferences that may or may not be shared across profiles.
