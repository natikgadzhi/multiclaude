package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"

	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	"github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Save the current Claude Code session as a named profile",
	Long: `Save the current Claude Code session as a named profile.

Captures the current credentials from the OS keychain and saves them
under the given profile name.

With --login, logs out of Claude Code first and opens a new login flow
so you can authenticate a different account, then saves it as the profile.`,
	Example: `  # Save current session
  multiclaude add work

  # Log into a new account and save it
  multiclaude add personal --login`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().Bool("set-default", false, "Set this profile as the default")
	addCmd.Flags().Bool("login", false, "Log out and start a new login flow before saving")
}

func runAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

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

	if store.Exists(name) {
		return errors.Wrap(
			fmt.Errorf("profile %q already exists", name),
			fmt.Sprintf("Profile %q already exists", name),
			fmt.Sprintf("Use a different name, or remove the existing one first:\n  multiclaude remove %s", name),
		)
	}

	ch := claude.NewClaudeHome(cfg.ClaudeHome)

	// Handle --login: log out, prompt for new login, then capture.
	login, _ := cmd.Flags().GetBool("login")
	if login {
		if err := doLoginFlow(cmd, ch); err != nil {
			return err
		}
	}

	// Read current credentials from Claude Code's keychain entry.
	creds, err := ch.ReadCredentials()
	if err != nil {
		return errors.Wrap(
			err,
			"No active Claude Code session found",
			"Log into Claude Code first, then run 'multiclaude add' again.",
		)
	}

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

	settings, err := ch.ReadSettings()
	if err != nil {
		debug.Log("no settings found, using empty: %v", err)
		settings = make(map[string]any)
	}

	if err := store.Create(name, creds, settings, label); err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}

	profiles, _ := store.List()
	if len(profiles) == 1 {
		debug.Log("first profile; marking %q as active", name)
		if err := store.WriteActive(name); err != nil {
			debug.Log("failed to write active state: %v", err)
		}
	}

	setDefault, _ := cmd.Flags().GetBool("set-default")
	if setDefault {
		cfg.DefaultProfile = name
		cfgPath, _ := cmd.Flags().GetString("config")
		if err := config.Save(cfgPath, cfg); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: profile created but could not update config: %v\n", err)
		} else {
			debug.Log("set default profile to %q", name)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Added profile: %s (%s)\n", name, label)
	return nil
}

// doLoginFlow handles the --login flag: logs out of Claude Code and
// guides the user through authenticating a new account.
func doLoginFlow(cmd *cobra.Command, ch *claude.ClaudeHome) error {
	// Check if claude is available.
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.Wrap(
			err,
			"Claude Code CLI not found in PATH",
			"Install Claude Code first: https://docs.anthropic.com/en/docs/claude-code",
		)
	}

	// Save current credentials if a profile is active, so we don't lose them.
	fmt.Fprintln(cmd.ErrOrStderr(), "Logging out of Claude Code...")
	logoutCmd := exec.Command("claude", "auth", "logout")
	logoutCmd.Stdout = os.Stdout
	logoutCmd.Stderr = os.Stderr
	if err := logoutCmd.Run(); err != nil {
		debug.Log("logout failed (may already be logged out): %v", err)
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "")
	fmt.Fprintln(cmd.ErrOrStderr(), "Please log into your new Claude Code account:")
	fmt.Fprintln(cmd.ErrOrStderr(), "  Run: claude auth login")
	fmt.Fprintln(cmd.ErrOrStderr(), "")
	fmt.Fprint(cmd.ErrOrStderr(), "Press Enter once you've logged in... ")

	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')

	// Verify credentials now exist.
	if _, err := ch.ReadCredentials(); err != nil {
		return errors.Wrap(
			err,
			"No credentials found after login",
			"Make sure you completed the login flow: claude auth login",
		)
	}

	return nil
}
