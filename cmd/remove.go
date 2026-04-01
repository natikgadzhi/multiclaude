package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile and its keychain entries",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	removeCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	// Check the profile exists.
	if !store.Exists(name) {
		return fmt.Errorf("profile %q does not exist", name)
	}

	// Cannot remove the active profile.
	activeName, _ := store.ActiveProfileName()
	if activeName == name {
		return fmt.Errorf("cannot remove the active profile %q. Switch to a different profile first with 'multiclaude use <name>'", name)
	}

	// Confirm unless --yes is passed.
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Fprintf(os.Stderr, "Remove profile %q? This cannot be undone. [y/N] ", name)
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := store.Delete(name); err != nil {
		return fmt.Errorf("removing profile: %w", err)
	}

	fmt.Printf("Removed profile: %s\n", name)
	return nil
}
