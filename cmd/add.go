package cmd

import (
	"fmt"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Save the current Claude Code session as a named profile",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdd,
}

func init() {
	addCmd.Flags().Bool("set-default", false, "Set this profile as the default")
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate profile name.
	if err := validateProfileName(name); err != nil {
		return err
	}

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	// Check profile doesn't already exist.
	if store.Exists(name) {
		return errors.Wrap(
			fmt.Errorf("profile %q already exists", name),
			fmt.Sprintf("Profile %q already exists", name),
			fmt.Sprintf("Use a different name, or remove the existing one first:\n  multiclaude remove %s", name),
		)
	}

	// Read current credentials from Claude Code's keychain entry.
	ch := claude.NewClaudeHome(cfg.ClaudeHome)
	creds, err := ch.ReadCredentials()
	if err != nil {
		return errors.Wrap(
			err,
			"No active Claude Code session found",
			"Log into Claude Code first, then run 'multiclaude add' again.",
		)
	}

	// Extract account info from credentials.
	info, err := ch.ActiveAccountInfo()
	if err != nil {
		return errors.Wrap(
			err,
			"Could not read account info from credentials",
			"Ensure you are logged into Claude Code with a valid account.",
		)
	}
	label := info.Label()

	debug.Log("adding profile %q for %s", name, label)

	// Read current settings (may not exist).
	settings, err := ch.ReadSettings()
	if err != nil {
		debug.Log("no settings found, using empty: %v", err)
		settings = make(map[string]any)
	}

	// Create the profile.
	if err := store.Create(name, creds, settings, label); err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}

	// Check if this is the first profile and mark it active.
	profiles, _ := store.List()
	if len(profiles) == 1 {
		debug.Log("first profile; marking %q as active", name)
		if err := store.WriteActive(name); err != nil {
			debug.Log("failed to write active state: %v", err)
		}
	}

	// Handle --set-default.
	setDefault, _ := cmd.Flags().GetBool("set-default")
	if setDefault {
		cfg.DefaultProfile = name
		cfgPath, _ := cmd.Flags().GetString("config")
		if err := config.Save(cfgPath, cfg); err != nil {
			// Non-fatal: profile was still created successfully.
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: profile created but could not update config: %v\n", err)
		} else {
			debug.Log("set default profile to %q", name)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added profile: %s (%s)\n", name, label)
	return nil
}
