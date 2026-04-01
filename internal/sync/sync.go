// Package sync manages which Claude Code settings are shared across profiles
// versus isolated per account. During a profile switch, shared fields from the
// outgoing profile's settings are merged into the incoming profile, while
// account-specific fields remain untouched.
package sync

// sharedFields lists settings.json keys that represent user preferences and
// should be synchronized across all profiles when switching.
//
// These are derived from Claude Code's actual settings.json structure:
//   - "permissions"           — tool permission rules (allow/deny lists)
//   - "model"                 — preferred model selection
//   - "hooks"                 — session lifecycle hooks configuration
//   - "enabledPlugins"        — which plugins/skills are enabled
//   - "extraKnownMarketplaces"— custom skill marketplace sources
//   - "effortLevel"           — reasoning effort preference
//   - "voiceEnabled"          — voice input toggle
//   - "skipDangerousModePermissionPrompt" — UI preference
//   - "skipAutoPermissionPrompt"          — UI preference
var sharedFields = []string{
	"permissions",
	"model",
	"hooks",
	"enabledPlugins",
	"extraKnownMarketplaces",
	"effortLevel",
	"voiceEnabled",
	"skipDangerousModePermissionPrompt",
	"skipAutoPermissionPrompt",
}

// accountFields lists settings.json keys that are tied to a specific account
// and must never be overwritten during a profile switch.
//
// When in doubt, a field should be classified as account-specific (safer).
var accountFields = []string{
	"env",
	"oauthAccount",
	"organizationId",
	"userId",
	"apiKey",
	"apiBaseUrl",
}

// SharedFields returns the list of settings.json keys that sync across profiles.
func SharedFields() []string {
	out := make([]string, len(sharedFields))
	copy(out, sharedFields)
	return out
}

// AccountFields returns the list of settings.json keys that are account-specific.
func AccountFields() []string {
	out := make([]string, len(accountFields))
	copy(out, accountFields)
	return out
}

// MergeSettings copies shared fields from src into dst, preserving dst's
// account-specific fields. Any key in src that is not in the shared list is
// ignored. Any key in dst that is not in the shared list is preserved as-is.
//
// The returned map is a new map — neither src nor dst are modified.
func MergeSettings(src, dst map[string]any) map[string]any {
	result := make(map[string]any, len(dst)+len(sharedFields))

	// Start with everything from dst.
	for k, v := range dst {
		result[k] = v
	}

	// Overwrite shared fields with values from src.
	shared := sharedSet()
	for k, v := range src {
		if shared[k] {
			result[k] = v
		}
	}

	return result
}

// sharedSet returns a set (map[string]bool) of shared field names for O(1) lookup.
func sharedSet() map[string]bool {
	set := make(map[string]bool, len(sharedFields))
	for _, f := range sharedFields {
		set[f] = true
	}
	return set
}
