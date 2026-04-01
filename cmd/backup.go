package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Create and manage profile backups",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a backup snapshot of all profiles",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all backup snapshots",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore <name>",
	Short: "Restore profiles from a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a backup snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}

func init() {
	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupRestoreCmd)
	backupCmd.AddCommand(backupDeleteCmd)
}
