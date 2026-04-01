package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a profile",
	Args:  cobra.ExactArgs(2),
	RunE:  runRename,
}

func runRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	// Validate new name.
	if err := validateProfileName(newName); err != nil {
		return err
	}

	// Validate old exists.
	if !store.Exists(oldName) {
		profiles, _ := store.List()
		if len(profiles) == 0 {
			return errors.Wrap(
				fmt.Errorf("profile %q not found", oldName),
				fmt.Sprintf("Profile %q not found", oldName),
				"No profiles exist yet. Run 'multiclaude add <name>' to create one.",
			)
		}
		names := make([]string, len(profiles))
		for i, p := range profiles {
			names[i] = p.Name
		}
		return errors.Wrap(
			fmt.Errorf("profile %q not found", oldName),
			fmt.Sprintf("Profile %q not found", oldName),
			fmt.Sprintf("Available profiles: %s", strings.Join(names, ", ")),
		)
	}

	// Validate new doesn't exist.
	if store.Exists(newName) {
		return errors.Wrap(
			fmt.Errorf("profile %q already exists", newName),
			fmt.Sprintf("Profile %q already exists", newName),
			fmt.Sprintf("Use a different name, or remove the existing one first:\n  multiclaude remove %s", newName),
		)
	}

	// Check if we're renaming the active profile so we can update the active file.
	activeName, _ := store.ActiveProfileName()
	isActive := activeName == oldName

	// Perform the rename (directory + keychain).
	if err := store.Rename(oldName, newName); err != nil {
		return errors.Wrap(
			err,
			fmt.Sprintf("Failed to rename profile %q to %q", oldName, newName),
			"Run 'multiclaude doctor' to check for issues.",
		)
	}

	// If we renamed the active profile, update the active file.
	if isActive {
		if err := store.WriteActive(newName); err != nil {
			// Non-fatal: rename succeeded but active tracking is stale.
			fmt.Fprintf(os.Stderr, "Warning: profile renamed but could not update active state: %v\n", err)
		}
	}

	fmt.Printf("Renamed: %s \u2192 %s\n", oldName, newName)
	return nil
}
