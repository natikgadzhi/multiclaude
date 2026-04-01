// Package keychain provides credential storage for multiclaude profiles
// using the OS keychain via go-keyring.
package keychain

import (
	"encoding/base64"
	"fmt"

	"github.com/zalando/go-keyring"
)

// ServicePrefix is the prefix used for all multiclaude keychain entries.
const ServicePrefix = "multiclaude"

// keychainAccount is the account name used for all credential entries.
const keychainAccount = "credentials"

// serviceName returns the keychain service name for a given profile.
// Format: multiclaude/{profileName}/oauth
func serviceName(profileName string) string {
	return fmt.Sprintf("%s/%s/oauth", ServicePrefix, profileName)
}

// StoreCredentials saves raw credential JSON bytes to the keychain for a profile.
// The bytes are base64-encoded before storage since the keyring API operates on strings.
func StoreCredentials(profileName string, creds []byte) error {
	encoded := base64.StdEncoding.EncodeToString(creds)
	return keyring.Set(serviceName(profileName), keychainAccount, encoded)
}

// GetCredentials retrieves raw credential JSON bytes from the keychain.
// Returns the decoded bytes from the base64-encoded keyring entry.
func GetCredentials(profileName string) ([]byte, error) {
	encoded, err := keyring.Get(serviceName(profileName), keychainAccount)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(encoded)
}

// DeleteCredentials removes a profile's credentials from the keychain.
func DeleteCredentials(profileName string) error {
	return keyring.Delete(serviceName(profileName), keychainAccount)
}

// HasCredentials checks if credentials exist for a profile.
func HasCredentials(profileName string) bool {
	_, err := keyring.Get(serviceName(profileName), keychainAccount)
	return err == nil
}
