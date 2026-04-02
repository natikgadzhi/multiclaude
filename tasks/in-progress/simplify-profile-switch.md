# Task: Simplify Profile Switching — Credentials Only

## Objective

Validate and enforce that profile switching only swaps authentication credentials,
NOT settings or other files in `~/.claude/`.

## Background

Currently, `store.SaveState` and `store.RestoreState` copy both credentials (via
keychain) and `settings.json` into/out of `~/.claude/`. The intent of the simplification
is that profiles are account identities — not settings bundles. Settings should stay
as the user left them and not be managed per-profile.

## Acceptance Criteria

- `multiclaude use <name>` writes only `.credentials.json` into `~/.claude/`
- `settings.json` and any other files in `~/.claude/` are NOT touched on switch
- Profile directories no longer store or restore `settings.json`
- `store.SaveState` / `store.RestoreState` only handle credentials
- Tests confirm settings are unchanged after a profile switch
- README and docs updated to reflect credentials-only switching

## What to change

- `internal/profile/profile.go` — `SaveState`, `RestoreState`: remove settings copy logic
- `internal/profile/profile.go` — `Create`: stop capturing `settings.json` on add
- `cmd/add.go` — stop passing settings into `store.Create` (or remove that parameter)
- Tests: update to assert settings are NOT overwritten on switch
- README: update "What multiclaude does" and "Switching flow" sections
