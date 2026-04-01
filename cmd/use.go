package cmd

import (
	"fmt"
	"strings"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/progress"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch to a named profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runUse,
}

func runUse(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	// Validate profile exists.
	if !store.Exists(name) {
		profiles, _ := store.List()
		if len(profiles) == 0 {
			return fmt.Errorf("profile %q does not exist. No profiles found — run 'multiclaude add <name>' to create one", name)
		}
		names := make([]string, len(profiles))
		for i, p := range profiles {
			names[i] = p.Name
		}
		return fmt.Errorf("profile %q does not exist. Available profiles: %s", name, strings.Join(names, ", "))
	}

	// Check if already active.
	active, _ := store.ActiveProfileName()
	if active == name {
		fmt.Fprintf(cmd.OutOrStdout(), "Already on profile: %s\n", name)
		return nil
	}

	format := output.Resolve(cmd)

	// Show spinner during switch.
	spin := progress.NewSpinner("Switching profile...", format)
	spin.Start()

	// Auto-save current state if there's an active profile.
	if active != "" {
		debug.Log("auto-saving state for current profile %q", active)
		if err := store.SaveState(active); err != nil {
			debug.Log("warning: could not save current profile state: %v", err)
		}
	}

	// Restore target profile's state.
	debug.Log("restoring state for target profile %q", name)
	if err := store.RestoreState(name); err != nil {
		spin.Finish()
		return fmt.Errorf("switching to profile %q: %w", name, err)
	}

	// Update active profile state file.
	if err := store.WriteActive(name); err != nil {
		spin.Finish()
		return fmt.Errorf("updating active profile state: %w", err)
	}

	spin.Finish()

	// Look up email for the success message.
	p, err := store.Get(name)
	if err != nil {
		// Profile was switched but we can't read metadata — still report success.
		fmt.Fprintf(cmd.OutOrStdout(), "Switched to profile: %s\n", name)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Switched to profile: %s (%s)\n", name, p.Email)
	return nil
}
