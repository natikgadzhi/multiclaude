package cmd

import (
	"fmt"
	"os"

	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/progress"
	"github.com/natikgadzhi/cli-kit/table"
	"github.com/natikgadzhi/multiclaude/internal/backup"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

// newBackupManager builds a backup.Manager from the default config paths.
func newBackupManager() (*backup.Manager, error) {
	profilesDir, err := config.ProfilesDir()
	if err != nil {
		return nil, fmt.Errorf("resolving profiles dir: %w", err)
	}

	backupsDir, err := config.BackupsDir()
	if err != nil {
		return nil, fmt.Errorf("resolving backups dir: %w", err)
	}

	cfg, err := config.Load("~/.config/multiclaude/config.toml")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	ch := claude.NewClaudeHome(cfg.ClaudeHome)
	store := profile.NewStore(profilesDir, ch)
	return backup.NewManager(backupsDir, store), nil
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create and manage profile backups",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a backup snapshot of all profiles",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format := output.Resolve(cmd)

		mgr, err := newBackupManager()
		if err != nil {
			return err
		}

		sp := progress.NewSpinner("Creating backup...", format)
		sp.Start()

		if err := mgr.Create(name); err != nil {
			sp.Finish()
			return err
		}

		sp.Finish()
		fmt.Fprintf(os.Stderr, "Backup %q created successfully.\n", name)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backup snapshots",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newBackupManager()
		if err != nil {
			return err
		}

		backups, err := mgr.List()
		if err != nil {
			return err
		}

		if len(backups) == 0 {
			fmt.Fprintln(os.Stderr, "No backups found.")
			return nil
		}

		format := output.Resolve(cmd)
		if output.IsJSON(format) {
			return output.PrintJSON(backups)
		}

		t := table.New()
		t.Header("Name", "Created", "Profiles")
		for _, b := range backups {
			t.Row(
				b.Name,
				b.CreatedAt.Local().Format("2006-01-02 15:04"),
				fmt.Sprintf("%d", b.Profiles),
			)
		}
		return t.Flush()
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore profiles from a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		format := output.Resolve(cmd)

		mgr, err := newBackupManager()
		if err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Warning: restoring will replace all current profiles.")

		sp := progress.NewSpinner("Restoring backup...", format)
		sp.Start()

		if err := mgr.Restore(name); err != nil {
			sp.Finish()
			return err
		}

		sp.Finish()
		fmt.Fprintf(os.Stderr, "Backup %q restored successfully.\n", name)
		return nil
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		mgr, err := newBackupManager()
		if err != nil {
			return err
		}

		if err := mgr.Delete(name); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Backup %q deleted.\n", name)
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
}
