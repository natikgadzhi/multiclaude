package cmd

import (
	"fmt"

	"github.com/natikgadzhi/cli-kit/config"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	mcconfig "github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

// loadConfig reads the multiclaude config file from the path given by --config flag.
func loadConfig(cmd *cobra.Command) (*mcconfig.Config, error) {
	cfgPath, _ := cmd.Flags().GetString("config")
	if cfgPath == "" {
		cfgPath = config.DefaultPath("multiclaude")
	}
	cfg, err := mcconfig.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// newProfileStore builds a profile.Store from the current config.
func newProfileStore(cfg *mcconfig.Config) (*profile.Store, error) {
	profilesDir, err := mcconfig.ProfilesDir()
	if err != nil {
		return nil, fmt.Errorf("resolving profiles directory: %w", err)
	}
	ch := claude.NewClaudeHome(cfg.ClaudeHome)
	return profile.NewStore(profilesDir, ch), nil
}
