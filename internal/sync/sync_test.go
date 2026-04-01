package sync

import (
	"testing"
)

func TestSharedFields(t *testing.T) {
	fields := SharedFields()
	if len(fields) == 0 {
		t.Fatal("SharedFields() returned empty slice")
	}

	// Verify it returns a copy, not the original slice.
	fields[0] = "tampered"
	original := SharedFields()
	if original[0] == "tampered" {
		t.Error("SharedFields() returned the original slice, not a copy")
	}
}

func TestAccountFields(t *testing.T) {
	fields := AccountFields()
	if len(fields) == 0 {
		t.Fatal("AccountFields() returned empty slice")
	}

	// Verify it returns a copy.
	fields[0] = "tampered"
	original := AccountFields()
	if original[0] == "tampered" {
		t.Error("AccountFields() returned the original slice, not a copy")
	}
}

func TestMergeSettings(t *testing.T) {
	tests := []struct {
		name      string
		src       map[string]any
		dst       map[string]any
		checkFunc func(t *testing.T, result map[string]any)
	}{
		{
			name: "shared fields copied from src to dst",
			src: map[string]any{
				"model":       "opus",
				"effortLevel": "high",
			},
			dst: map[string]any{
				"model":       "sonnet",
				"effortLevel": "low",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
				if result["effortLevel"] != "high" {
					t.Errorf("effortLevel = %v, want %q", result["effortLevel"], "high")
				}
			},
		},
		{
			name: "account fields preserved in dst",
			src: map[string]any{
				"env":            map[string]any{"SHELL": "/bin/bash"},
				"organizationId": "org-from-src",
				"model":          "opus",
			},
			dst: map[string]any{
				"env":            map[string]any{"SHELL": "/bin/zsh"},
				"organizationId": "org-from-dst",
				"model":          "sonnet",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				// Account-specific fields stay from dst.
				env := result["env"].(map[string]any)
				if env["SHELL"] != "/bin/zsh" {
					t.Errorf("env.SHELL = %v, want %q", env["SHELL"], "/bin/zsh")
				}
				if result["organizationId"] != "org-from-dst" {
					t.Errorf("organizationId = %v, want %q", result["organizationId"], "org-from-dst")
				}
				// Shared field comes from src.
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
			},
		},
		{
			name: "missing shared fields in src do not delete from dst",
			src:  map[string]any{},
			dst: map[string]any{
				"model":       "sonnet",
				"effortLevel": "low",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "sonnet" {
					t.Errorf("model = %v, want %q", result["model"], "sonnet")
				}
				if result["effortLevel"] != "low" {
					t.Errorf("effortLevel = %v, want %q", result["effortLevel"], "low")
				}
			},
		},
		{
			name: "new shared fields from src added to dst",
			src: map[string]any{
				"model":        "opus",
				"voiceEnabled": true,
			},
			dst: map[string]any{
				"model": "sonnet",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
				if result["voiceEnabled"] != true {
					t.Errorf("voiceEnabled = %v, want true", result["voiceEnabled"])
				}
			},
		},
		{
			name: "unknown fields in src are ignored",
			src: map[string]any{
				"model":          "opus",
				"unknownFeature": "should-not-copy",
			},
			dst: map[string]any{
				"model": "sonnet",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
				if _, exists := result["unknownFeature"]; exists {
					t.Error("unknownFeature should not be copied to result")
				}
			},
		},
		{
			name: "unknown fields in dst are preserved",
			src: map[string]any{
				"model": "opus",
			},
			dst: map[string]any{
				"model":          "sonnet",
				"customSetting":  "keep-this",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
				if result["customSetting"] != "keep-this" {
					t.Errorf("customSetting = %v, want %q", result["customSetting"], "keep-this")
				}
			},
		},
		{
			name: "both maps empty",
			src:  map[string]any{},
			dst:  map[string]any{},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if len(result) != 0 {
					t.Errorf("result has %d keys, want 0", len(result))
				}
			},
		},
		{
			name: "nil src treated as empty",
			src:  nil,
			dst: map[string]any{
				"model":          "sonnet",
				"organizationId": "org-123",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "sonnet" {
					t.Errorf("model = %v, want %q", result["model"], "sonnet")
				}
				if result["organizationId"] != "org-123" {
					t.Errorf("organizationId = %v, want %q", result["organizationId"], "org-123")
				}
			},
		},
		{
			name: "nil dst treated as empty",
			src: map[string]any{
				"model": "opus",
			},
			dst: nil,
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				if result["model"] != "opus" {
					t.Errorf("model = %v, want %q", result["model"], "opus")
				}
			},
		},
		{
			name: "complex shared field overwritten entirely",
			src: map[string]any{
				"permissions": map[string]any{
					"allow": []any{"Bash(git *)", "Bash(go *)"},
				},
			},
			dst: map[string]any{
				"permissions": map[string]any{
					"allow": []any{"Bash(npm *)"},
				},
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				perms := result["permissions"].(map[string]any)
				allow := perms["allow"].([]any)
				if len(allow) != 2 {
					t.Errorf("permissions.allow has %d entries, want 2", len(allow))
				}
			},
		},
		{
			name: "does not mutate src or dst",
			src: map[string]any{
				"model": "opus",
			},
			dst: map[string]any{
				"model":          "sonnet",
				"organizationId": "org-dst",
			},
			checkFunc: func(t *testing.T, result map[string]any) {
				t.Helper()
				// This is checked in the test body below, not here.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeSettings(tt.src, tt.dst)
			tt.checkFunc(t, result)
		})
	}

	// Dedicated mutation check.
	t.Run("MergeSettings does not mutate inputs", func(t *testing.T) {
		src := map[string]any{"model": "opus"}
		dst := map[string]any{"model": "sonnet", "organizationId": "org-dst"}

		_ = MergeSettings(src, dst)

		if src["model"] != "opus" {
			t.Error("src was mutated")
		}
		if dst["model"] != "sonnet" {
			t.Error("dst was mutated")
		}
		if dst["organizationId"] != "org-dst" {
			t.Error("dst was mutated")
		}
	})
}

func TestSharedAndAccountFieldsDoNotOverlap(t *testing.T) {
	shared := make(map[string]bool)
	for _, f := range SharedFields() {
		shared[f] = true
	}
	for _, f := range AccountFields() {
		if shared[f] {
			t.Errorf("field %q appears in both SharedFields and AccountFields", f)
		}
	}
}
