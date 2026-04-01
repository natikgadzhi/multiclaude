package cmd

import (
	"fmt"
	"os"

	"github.com/natikgadzhi/multiclaude/internal/claude"
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

	// Validate old exists.
	if !store.Exists(oldName) {
		return fmt.Errorf("profile %q does not exist", oldName)
	}

	// Validate new doesn't exist.
	if store.Exists(newName) {
		return fmt.Errorf("profile %q already exists", newName)
	}

	// Check if we're renaming the active profile so we can update the symlink.
	activeName, _ := store.ActiveProfileName()
	isActive := activeName == oldName

	// Perform the rename (directory + keychain).
	if err := store.Rename(oldName, newName); err != nil {
		return fmt.Errorf("renaming profile: %w", err)
	}

	// If we renamed the active profile, update the symlink.
	if isActive {
		ch := claude.NewClaudeHome(cfg.ClaudeHome)
		if ch.IsSymlink() {
			// Remove the old symlink and create a new one pointing to the renamed profile.
			if err := os.Remove(ch.Path); err != nil {
				return fmt.Errorf("removing old symlink: %w", err)
			}
			newTarget := store.ProfileDir(newName)
			if err := os.Symlink(newTarget, ch.Path); err != nil {
				return fmt.Errorf("creating updated symlink: %w", err)
			}
		}
	}

	fmt.Printf("Renamed: %s \u2192 %s\n", oldName, newName)
	return nil
}
