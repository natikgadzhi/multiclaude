package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var renameCmd = &cobra.Command{
	Use:   "rename <old> <new>",
	Short: "Rename a profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "not implemented yet")
		return nil
	},
}
