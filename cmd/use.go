package cmd

import (
	"fmt"
	"strings"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
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
			return errors.Wrap(
				fmt.Errorf("profile %q not found", name),
				fmt.Sprintf("Profile %q not found", name),
				"No profiles exist yet. Log into Claude Code first, then run:\n  multiclaude add <name>",
			)
		}
		names := make([]string, len(profiles))
		for i, p := range profiles {
			names[i] = p.Name
		}
		return errors.Wrap(
			fmt.Errorf("profile %q not found", name),
			fmt.Sprintf("Profile %q not found", name),
			fmt.Sprintf("Available profiles: %s\nSwitch with: multiclaude use <name>", strings.Join(names, ", ")),
		)
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
		return errors.Wrap(
			err,
			fmt.Sprintf("Failed to switch to profile %q", name),
			"The profile's credentials may be missing from the keychain.\nTry: multiclaude doctor",
		)
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
