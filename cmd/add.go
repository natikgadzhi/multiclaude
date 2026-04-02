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
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Log into a Claude Code account and save it as a named profile",
	Long: `Log into a Claude Code account and save it as a named profile.

Logs out of the current Claude Code session, opens a new login flow so
you can authenticate a different account, then saves those credentials
under the given profile name.`,
	Example: `  multiclaude add work
  multiclaude add personal`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return fmt.Errorf(`profile name required

multiclaude add logs into a new Claude Code account and saves it as a named profile.

  multiclaude add work
  multiclaude add personal`)
		}
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: runAdd,
}

func init() {
	addCmd.Flags().Bool("set-default", false, "Set this profile as the default")
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

	// Save the current active profile's state before logout so we don't lose
	// any refreshed tokens since the last switch.
	active, _ := store.ActiveProfileName()
	if active != "" {
		debug.Log("saving state for current profile %q before logout", active)
		if err := store.SaveState(active); err != nil {
			debug.Log("warning: could not save current profile state: %v", err)
		}
	}

	// Log out and prompt for a new login before capturing credentials.
	if err := doLoginFlow(cmd); err != nil {
		return err
	}

	// Step 1: claude auth status is the source of truth for the new account.
	authStatus := getClaudeAuthStatus()
	if authStatus == nil || !authStatus.LoggedIn {
		return errors.Wrap(
			fmt.Errorf("claude auth status reports not logged in"),
			"No active Claude Code session found after login",
			"Make sure you completed the login flow:\n  claude auth login\nThen try again:\n  multiclaude add "+name,
		)
	}

	debug.Log("claude auth status: email=%s org=%s plan=%s",
		authStatus.Email, authStatus.OrgName, authStatus.SubscriptionType)

	// Step 2: Read the keychain credentials and verify they match.
	creds, err := ch.ReadCredentials()
	if err != nil {
		return errors.Wrap(
			err,
			"No credentials found in keychain",
			"Claude Code may not have written credentials yet. Try logging in again:\n  claude auth login",
		)
	}

	// Parse the email from the keychain credentials to cross-check.
	info, err := ch.ActiveAccountInfo()
	if err != nil {
		return errors.Wrap(
			err,
			"Could not read account info from credentials",
			"Ensure you are logged into Claude Code with a valid account.",
		)
	}

	if info.Email == "" || authStatus.Email == "" {
		debug.Log("warning: cannot fully verify email cross-check (keychain=%q auth=%q); proceeding",
			info.Email, authStatus.Email)
	} else if info.Email != authStatus.Email {
		return errors.Wrap(
			fmt.Errorf("credential mismatch: keychain has %q but auth status reports %q", info.Email, authStatus.Email),
			"Credential mismatch after login",
			"The keychain still contains stale credentials from a previous account.\n"+
				"Try logging out and in again:\n  claude auth logout\n  claude auth login\n"+
				"Then run:\n  multiclaude add "+name,
		)
	} else {
		debug.Log("credentials verified: keychain email %q matches auth status", info.Email)
	}

	opts := profile.CreateOptions{
		Email:            authStatus.Email,
		OrgName:          authStatus.OrgName,
		SubscriptionType: authStatus.SubscriptionType,
	}

	settings, err := ch.ReadSettings()
	if err != nil {
		debug.Log("no settings found, using empty: %v", err)
		settings = make(map[string]any)
	}

	if err := store.Create(name, creds, settings, opts); err != nil {
		return fmt.Errorf("creating profile: %w", err)
	}

	debug.Log("marking %q as active", name)
	if err := store.WriteActive(name); err != nil {
		debug.Log("failed to write active state: %v", err)
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

	label := authStatus.Email
	if authStatus.SubscriptionType != "" {
		label = fmt.Sprintf("%s (%s)", authStatus.Email, authStatus.SubscriptionType)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Added profile: %s (%s)\n", name, label)
	return nil
}

// doLoginFlow logs out of Claude Code and guides the user through
// authenticating a new account.
func doLoginFlow(cmd *cobra.Command) error {
	if _, err := exec.LookPath("claude"); err != nil {
		return errors.Wrap(
			err,
			"Claude Code CLI not found in PATH",
			"Install Claude Code first: https://docs.anthropic.com/en/docs/claude-code",
		)
	}

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

	return nil
}
