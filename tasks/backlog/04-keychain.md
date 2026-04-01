# Task 04: Keychain Operations

## Objective
Store and retrieve Claude OAuth credentials in the OS keychain, keyed per profile.

## Acceptance Criteria
- `internal/keychain/keychain.go` with:
  ```go
  const ServicePrefix = "multiclaude"

  // StoreCredentials saves a profile's OAuth credentials to the keychain.
  func StoreCredentials(profileName string, creds []byte) error

  // GetCredentials retrieves a profile's OAuth credentials from the keychain.
  func GetCredentials(profileName string) ([]byte, error)

  // DeleteCredentials removes a profile's credentials from the keychain.
  func DeleteCredentials(profileName string) error

  // HasCredentials checks if a profile has stored credentials.
  func HasCredentials(profileName string) bool
  ```
- Keychain service name: `multiclaude/{profileName}/oauth`
- Uses `cli-kit/auth` (go-keyring) under the hood — NOT raw subprocess calls
- Credentials stored as the raw JSON bytes from `.credentials.json`
- Tests use `keyring.MockInit()` for in-memory mock

## Dependencies
- Task 01 (bootstrap)
