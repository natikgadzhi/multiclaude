# Fix Keychain Auth Restoration & Enhance `current` Command

## Problem

### Bug 1: go-keyring encoding corrupts Claude Code's keychain entry

`zalando/go-keyring` v0.2.8 on macOS **always** base64-wraps values on `Set()`:

```go
// keyring_darwin.go:74
password = base64EncodingPrefix + base64.StdEncoding.EncodeToString([]byte(password))
```

When multiclaude calls `keyring.Set("Claude Code-credentials", ...)` in `WriteCredentials`, the keychain value becomes `go-keyring-base64:<base64(JSON)>` instead of the raw JSON that Claude Code (TypeScript/Node) expects. Claude Code reads the raw keychain value, can't parse it, and the user appears logged out.

**Read path is fine** — go-keyring's `Get()` transparently strips its own prefix. So multiclaude can always read what it or Claude Code wrote. The problem is only on write.

multiclaude's own keychain entries (`multiclaude/{name}/oauth`) are unaffected because both read and write go through go-keyring.

### Bug 2: `doLoginFlow` doesn't save active profile before logout

`cmd/add.go` runs `claude auth logout` without first saving the active profile's current credentials to multiclaude's keychain. If the user's access token was refreshed since the last `use` or `add`, the newer token is lost. On next switch, stale credentials are restored.

### Enhancement: `multiclaude current` lacks Claude auth context

Users can't tell whether the active multiclaude profile actually has working Claude auth. The command should show Claude Code's auth status alongside the profile name.

## Design

### 1. Bypass go-keyring for Claude Code's keychain entry

Replace `keyring.Get/Set` in `internal/claude/claude.go` with direct `/usr/bin/security` calls, **only** for Claude Code's own entry (`"Claude Code-credentials"`).

#### `ReadCredentials` implementation

Shell out to:
```
/usr/bin/security find-generic-password -s "Claude Code-credentials" -wa <username>
```

Handle backwards compatibility:
- If the returned string starts with `go-keyring-base64:`, strip the prefix and base64-decode (entry was previously corrupted by multiclaude).
- If the returned string starts with `go-keyring-encoded:`, strip the prefix and hex-decode (older go-keyring format).
- Otherwise, return as-is (entry was written by Claude Code or by the fixed multiclaude).

#### `WriteCredentials` implementation

1. Delete existing entry: `/usr/bin/security delete-generic-password -s "Claude Code-credentials" -a <username>` (ignore "not found" errors).
2. Add raw JSON: `/usr/bin/security add-generic-password -s "Claude Code-credentials" -a <username> -w <raw-json>`.

The raw JSON is passed via `-w` flag. Since credential JSON is single-line and ASCII, this is safe. Use `shellescape` or pass via stdin if needed for safety.

#### Platform considerations

- macOS: use `/usr/bin/security` directly as described above.
- Linux/Windows: go-keyring doesn't add the base64 prefix on those platforms (it uses D-Bus Secret Service / Windows Credential Manager natively). Keep using go-keyring there. Add a build tag or runtime `GOOS` check.
- For now, implement macOS only (primary platform). Add a `// TODO: verify go-keyring behavior on Linux` comment.

#### Files changed

- `internal/claude/claude.go` — replace `ReadCredentials` and `WriteCredentials` bodies. Remove `keyring` import (only used for Claude Code's entry). Add `os/exec`, `encoding/base64`, `encoding/hex`, `strings` imports.

### 2. Save active profile before logout in `add`

In `cmd/add.go`, before `doLoginFlow` is called, check for an active profile and save its state:

```go
active, _ := store.ActiveProfileName()
if active != "" {
    debug.Log("saving state for current profile %q before logout", active)
    if err := store.SaveState(active); err != nil {
        debug.Log("warning: could not save current profile state: %v", err)
    }
}
```

This mirrors the pattern in `cmd/use.go` lines 69-74. The save is best-effort (logged warning, not fatal) because the user may not have valid credentials if they're already logged out.

#### Files changed

- `cmd/add.go` — add save-state block before `doLoginFlow` call. Requires passing `store` setup earlier (before `doLoginFlow`).

Note: `newProfileStore` and `loadConfig` are already called before `doLoginFlow`, so `store` is available. The active profile name check also works because `store.SetActiveFile` is called inside `newProfileStore`.

### 3. Enhance `multiclaude current` with Claude auth status

Run `claude auth status --json` and present the result alongside the profile name.

#### `claude auth status --json` output format

```json
{
  "loggedIn": true,
  "authMethod": "claude.ai",
  "apiProvider": "firstParty",
  "email": "user@example.com",
  "orgId": "...",
  "orgName": "...",
  "subscriptionType": "pro"
}
```

#### TTY output format

```
Profile:   work
Email:     user@example.com (Pro)
Claude:    authenticated
```

If Claude auth check fails or returns unauthenticated:
```
Profile:   work
Email:     user@example.com
Claude:    not authenticated
```

#### JSON output format

```json
{
  "name": "work",
  "email": "user@example.com",
  "claude_authenticated": true,
  "claude_subscription_type": "pro"
}
```

#### Implementation

- Run `exec.Command("claude", "auth", "status", "--json")` and parse stdout as JSON.
- If `claude` is not in PATH or the command fails, treat as "unknown" status (not an error).
- Add the auth status fields to the `currentOutput` struct.

#### Files changed

- `cmd/current.go` — add auth status check, expand `currentOutput` struct, update TTY and JSON output.

## Testing

- **Keychain read/write**: Hard to unit test directly (requires macOS keychain). Add integration test notes. The existing `keyring.MockInit()` tests won't cover this since the fix bypasses go-keyring.
- **Save-before-logout**: Existing test patterns in `cmd/` can verify `SaveState` is called.
- **Current command**: Mock the `claude auth status` call or test with a helper that captures the exec call.

## Migration / Backwards Compatibility

- Entries already corrupted by multiclaude (with `go-keyring-base64:` prefix) are handled transparently by the new `ReadCredentials`.
- After the fix, `WriteCredentials` writes raw JSON, so Claude Code can read it immediately.
- multiclaude's own entries stay in go-keyring format — no migration needed.
