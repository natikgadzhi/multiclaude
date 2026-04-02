package cmd

import (
	"fmt"
	"strings"

	"github.com/natikgadzhi/cli-kit/output"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the active profile",
	Args:  cobra.NoArgs,
	RunE:  runCurrent,
}

// currentOutput is the JSON-serializable representation of the active profile.
type currentOutput struct {
	Name                  string `json:"name"`
	Email                 string `json:"email,omitempty"`
	ClaudeAuthenticated   bool   `json:"claude_authenticated"`
	ClaudeSubscriptionType string `json:"claude_subscription_type,omitempty"`
}

func runCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	activeName, err := store.ActiveProfileName()
	if err != nil {
		return fmt.Errorf("determining active profile: %w", err)
	}

	if activeName == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "No active profile. Log into Claude Code first, then run:\n  multiclaude add <name>")
		return nil
	}

	// Check Claude Code's auth status.
	authStatus := getClaudeAuthStatus()

	result := currentOutput{
		Name: activeName,
	}
	if authStatus != nil {
		result.Email = authStatus.Email
		result.ClaudeAuthenticated = authStatus.LoggedIn
		if authStatus.LoggedIn {
			result.ClaudeSubscriptionType = authStatus.SubscriptionType
		}
	}

	format := output.Resolve(cmd)
	if output.IsJSON(format) {
		return output.PrintJSON(result)
	}

	// TTY output.
	fmt.Fprintf(cmd.OutOrStdout(), "Profile:   %s\n", result.Name)
	if result.Email != "" {
		if result.ClaudeSubscriptionType != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Email:     %s (%s)\n",
				result.Email, capitalizeFirst(result.ClaudeSubscriptionType))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Email:     %s\n", result.Email)
		}
	}
	if authStatus != nil {
		if result.ClaudeAuthenticated {
			fmt.Fprintln(cmd.OutOrStdout(), "Claude:    authenticated")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Claude:    not authenticated")
		}
	}

	return nil
}

// capitalizeFirst returns the string with its first letter capitalized.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
