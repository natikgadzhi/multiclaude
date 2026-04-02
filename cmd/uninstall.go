package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/keychain"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove all multiclaude state and keychain entries",
	Long: `Remove all multiclaude state from this machine.

This deletes:
  - All profile metadata under ~/.config/multiclaude/
  - All multiclaude keychain entries (one per profile)

This does NOT touch ~/.claude/ — your current Claude Code credentials
and settings remain intact.

If more than one profile exists, remove the extras first with
'multiclaude remove <name>', then re-run uninstall.`,
	Args: cobra.NoArgs,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.Flags().Bool("dry-run", false, "Print what would be deleted without doing it")
	uninstallCmd.Flags().BoolP("force", "f", false, "Proceed even if active profile is unclear")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("resolving config directory: %w", err)
	}

	return doUninstall(cmd.OutOrStdout(), os.Stdin, store, configDir, dryRun, force)
}

// doUninstall performs the uninstall logic. It is a standalone function so
// tests can inject an isolated store and config directory.
func doUninstall(out io.Writer, in io.Reader, store *profile.Store, configDir string, dryRun, force bool) error {
	profiles, err := store.List()
	if err != nil {
		return fmt.Errorf("listing profiles: %w", err)
	}

	// Refuse if more than one profile exists.
	if len(profiles) > 1 {
		names := make([]string, len(profiles))
		for i, p := range profiles {
			names[i] = p.Name
		}
		return errors.Wrap(
			fmt.Errorf("%d profiles exist", len(profiles)),
			fmt.Sprintf("%d profiles exist", len(profiles)),
			fmt.Sprintf("Remove extras first with 'multiclaude remove <name>', then re-run uninstall.\nProfiles: %s", strings.Join(names, ", ")),
		)
	}

	// If exactly one profile exists, check whether it is the active one.
	if len(profiles) == 1 && !profiles[0].IsActive {
		if !force {
			fmt.Fprintf(os.Stderr, "Warning: profile %q exists but is not active.\n", profiles[0].Name)
			fmt.Fprintf(os.Stderr, "~/.claude/ may not have valid credentials for this profile.\n")
			fmt.Fprintf(os.Stderr, "Proceed anyway? This will delete all multiclaude state. [y/N] ")
			reader := bufio.NewReader(in)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(os.Stderr, "Aborted.")
				return nil
			}
		} else {
			fmt.Fprintf(os.Stderr, "Warning: profile %q is not active. Proceeding with --force.\n", profiles[0].Name)
		}
	}

	if dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] Would delete keychain entries:\n")
		for _, p := range profiles {
			fmt.Fprintf(os.Stderr, "  - multiclaude/%s/oauth\n", p.Name)
		}
		fmt.Fprintf(os.Stderr, "[dry-run] Would delete directory: %s\n", configDir)
		return nil
	}

	// Delete all keychain entries.
	for _, p := range profiles {
		if err := keychain.DeleteCredentials(p.Name); err != nil {
			// Keychain entry may not exist — log and continue.
			fmt.Fprintf(os.Stderr, "Warning: could not delete keychain entry for %q: %v\n", p.Name, err)
		}
	}

	// Delete the entire multiclaude config directory.
	if err := os.RemoveAll(configDir); err != nil {
		return fmt.Errorf("removing config directory %s: %w", configDir, err)
	}

	fmt.Fprintf(out, "multiclaude state removed.\n")
	fmt.Fprintf(out, "Your Claude Code credentials in ~/.claude/ are untouched.\n")
	return nil
}
