package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

// validName matches alphanumeric characters and dashes, with no leading/trailing dashes.
var validName = regexp.MustCompile(`^[a-zA-Z0-9]+([a-zA-Z0-9-]*[a-zA-Z0-9]+)?$`)

// claudeAuthStatus represents the JSON output of `claude auth status --json`.
type claudeAuthStatus struct {
	LoggedIn         bool   `json:"loggedIn"`
	AuthMethod       string `json:"authMethod"`
	APIProvider      string `json:"apiProvider"`
	Email            string `json:"email"`
	OrgID            string `json:"orgId"`
	OrgName          string `json:"orgName"`
	SubscriptionType string `json:"subscriptionType"`
}

// getClaudeAuthStatus runs `claude auth status --json` and parses the result.
// Returns nil if the command fails or is not available.
func getClaudeAuthStatus() *claudeAuthStatus {
	cmd := exec.Command("claude", "auth", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		debug.Log("claude auth status failed: %v", err)
		return nil
	}

	var status claudeAuthStatus
	if err := json.Unmarshal(out, &status); err != nil {
		debug.Log("could not parse claude auth status output: %v", err)
		return nil
	}
	return &status
}

// loadConfig reads the multiclaude config from the --config flag path.
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("reading config flag: %w", err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, errors.Wrap(
			err,
			"Could not load configuration",
			"Check the TOML syntax in your config file. Minimal example:\n\n  # ~/.config/multiclaude/config.toml\n  default_profile = \"work\"\n  claude_home = \"~/.claude\"",
		)
	}
	return cfg, nil
}

// newProfileStore creates a profile Store from the loaded config,
// with active profile tracking via the state file.
func newProfileStore(cfg *config.Config) (*profile.Store, error) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return nil, fmt.Errorf("resolving profiles directory: %w", err)
	}

	ch := claude.NewClaudeHome(cfg.ClaudeHome)
	store := profile.NewStore(profilesDir, ch)

	activeFile, err := config.ActiveFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolving active file path: %w", err)
	}
	store.SetActiveFile(activeFile)

	return store, nil
}

// validateProfileName checks that a profile name contains only
// alphanumeric characters and dashes.
func validateProfileName(name string) error {
	if !validName.MatchString(name) {
		return errors.Wrap(
			fmt.Errorf("invalid profile name %q", name),
			fmt.Sprintf("Invalid profile name %q", name),
			"Profile names must contain only letters, numbers, and dashes.\nExamples: work, personal, my-org-1",
		)
	}
	return nil
}
