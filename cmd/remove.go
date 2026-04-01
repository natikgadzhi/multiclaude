package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile and its keychain entries",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}

func init() {
	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}
