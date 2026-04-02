package cmd

import (
	"fmt"

	"github.com/natikgadzhi/cli-kit/output"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active profile",
	Args:  cobra.NoArgs,
	RunE:  runCurrent,
}

// currentOutput is the JSON-serializable representation of the active profile.
type currentOutput struct {
	Name string `json:"name"`
}

func runCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	activeName, err := store.ActiveProfileName()
	if err != nil {
		return fmt.Errorf("determining active profile: %w", err)
	}

	if activeName == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No active profile. Log into Claude Code first, then run:\n  multiclaude add <name>")
		return nil
	}

	format := output.Resolve(cmd)
	if output.IsJSON(format) {
		return output.PrintJSON(currentOutput{Name: activeName})
	}

	fmt.Fprintln(cmd.OutOrStdout(), activeName)
	return nil
}
