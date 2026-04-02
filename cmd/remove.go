package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Delete a profile and its keychain entries",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf(`profile name required

  multiclaude remove <name>
  multiclaude list    — see available profiles`)
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
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
		profiles, _ := store.List()
		if len(profiles) == 0 {
			return errors.Wrap(
				fmt.Errorf("profile %q not found", name),
				fmt.Sprintf("Profile %q not found", name),
				"No profiles exist yet. Run 'multiclaude list' to see available profiles.",
			)
		}
		names := make([]string, len(profiles))
		for i, p := range profiles {
			names[i] = p.Name
		}
		return errors.Wrap(
			fmt.Errorf("profile %q not found", name),
			fmt.Sprintf("Profile %q not found", name),
			fmt.Sprintf("Available profiles: %s", strings.Join(names, ", ")),
		)
	}

	// Cannot remove the active profile.
	activeName, _ := store.ActiveProfileName()
	if activeName == name {
		profiles, _ := store.List()
		var others []string
		for _, p := range profiles {
			if p.Name != name {
				others = append(others, p.Name)
			}
		}
		suggestion := "Switch to another profile first."
		if len(others) > 0 {
			suggestion = fmt.Sprintf("Switch to another profile first:\n  multiclaude use %s", others[0])
		}
		return errors.Wrap(
			fmt.Errorf("cannot remove active profile %q", name),
			fmt.Sprintf("Cannot remove the active profile %q", name),
			suggestion,
		)
	}

	// Confirm unless --yes is passed.
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		fmt.Fprintf(os.Stderr, "Remove profile %q? This deletes multiclaude's saved credentials for this profile.\nYour Claude Code settings and ~/.claude/ are not affected. [y/N] ", name)
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
