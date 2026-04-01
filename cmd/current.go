package cmd

import (
	"fmt"

	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/table"
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
	Name  string `json:"name"`
	Email string `json:"email"`
}

// RenderTable implements output.TableRenderer for currentOutput.
func (c currentOutput) RenderTable(t *table.Table) {
	t.Header("Profile", "Email")
	t.Row(c.Name, c.Email)
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

	p, err := store.Get(activeName)
	if err != nil {
		return fmt.Errorf("reading active profile: %w", err)
	}

	format := output.Resolve(cmd)
	data := currentOutput{Name: p.Name, Email: p.Email}
	return output.Print(format, data, data)
}
