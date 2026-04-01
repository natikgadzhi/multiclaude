package keychain

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func TestStoreAndGetCredentials(t *testing.T) {
	keyring.MockInit()

	creds := []byte(`{"token":"abc123","refresh":"def456"}`)

	if err := StoreCredentials("work", creds); err != nil {
		t.Fatalf("StoreCredentials() error = %v", err)
	}

	got, err := GetCredentials("work")
	if err != nil {
		t.Fatalf("GetCredentials() error = %v", err)
	}

	if string(got) != string(creds) {
		t.Errorf("GetCredentials() = %q, want %q", got, creds)
	}
}

func TestGetCredentials_NotFound(t *testing.T) {
	keyring.MockInit()

	_, err := GetCredentials("nonexistent")
	if err == nil {
		t.Fatal("GetCredentials() expected error for nonexistent profile, got nil")
	}
}

func TestDeleteCredentials(t *testing.T) {
	keyring.MockInit()

	creds := []byte(`{"token":"abc123"}`)

	if err := StoreCredentials("temp", creds); err != nil {
		t.Fatalf("StoreCredentials() error = %v", err)
	}

	if err := DeleteCredentials("temp"); err != nil {
		t.Fatalf("DeleteCredentials() error = %v", err)
	}

	_, err := GetCredentials("temp")
	if err == nil {
		t.Fatal("GetCredentials() expected error after delete, got nil")
	}
}

func TestHasCredentials(t *testing.T) {
	keyring.MockInit()

	if HasCredentials("missing") {
		t.Error("HasCredentials() = true for missing profile, want false")
	}

	creds := []byte(`{"token":"abc123"}`)
	if err := StoreCredentials("present", creds); err != nil {
		t.Fatalf("StoreCredentials() error = %v", err)
	}

	if !HasCredentials("present") {
		t.Error("HasCredentials() = false for stored profile, want true")
	}
}

func TestServiceName(t *testing.T) {
	tests := []struct {
		profile string
		want    string
	}{
		{"work", "multiclaude/work/oauth"},
		{"personal", "multiclaude/personal/oauth"},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			got := serviceName(tt.profile)
			if got != tt.want {
				t.Errorf("serviceName(%q) = %q, want %q", tt.profile, got, tt.want)
			}
		})
	}
}
