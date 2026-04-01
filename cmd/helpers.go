package cmd

import (
	"fmt"
	"regexp"

	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

// validName matches alphanumeric characters and dashes, with no leading/trailing dashes.
var validName = regexp.MustCompile(`^[a-zA-Z0-9]+([a-zA-Z0-9-]*[a-zA-Z0-9]+)?$`)

// loadConfig reads the multiclaude config from the --config flag path.
func loadConfig(cmd *cobra.Command) (*config.Config, error) {
	cfgPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return nil, fmt.Errorf("reading config flag: %w", err)
	}
	return config.Load(cfgPath)
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
		return fmt.Errorf("invalid profile name %q: must contain only alphanumeric characters and dashes", name)
	}
	return nil
}
