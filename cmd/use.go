package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/progress"
	"github.com/natikgadzhi/multiclaude/internal/backup"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/profile"
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

	// Auto-backup before switching if enabled.
	if cfg.AutoBackup {
		autoBackupBeforeSwitch(cfg)
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

// autoBackupBeforeSwitch creates an auto-backup and prunes old ones.
// Failures are logged but do not block the switch.
func autoBackupBeforeSwitch(cfg *config.Config) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		debug.Log("auto-backup: could not resolve profiles dir: %v", err)
		return
	}

	backupsDir, err := config.BackupsDir()
	if err != nil {
		debug.Log("auto-backup: could not resolve backups dir: %v", err)
		return
	}

	ch := claude.NewClaudeHome(cfg.ClaudeHome)
	store := profile.NewStore(profilesDir, ch)
	mgr := backup.NewManager(backupsDir, store)

	name := "auto-" + time.Now().Format("2006-01-02T15-04-05")
	if err := mgr.Create(name); err != nil {
		debug.Log("auto-backup: create failed: %v", err)
		return
	}
	debug.Log("auto-backup created: %s", name)

	if err := mgr.PruneAutoBackups(5); err != nil {
		debug.Log("auto-backup: prune failed: %v", err)
	}
}
