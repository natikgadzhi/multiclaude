package cmd

import (
	"github.com/natikgadzhi/cli-kit/config"
	"github.com/natikgadzhi/cli-kit/debug"
	"github.com/natikgadzhi/cli-kit/errors"
	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/version"
	"github.com/spf13/cobra"
)

// Build-time variables set via ldflags.
var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

var versionInfo = &version.Info{
	Version: buildVersion,
	Commit:  buildCommit,
	Date:    buildDate,
}

var rootCmd = &cobra.Command{
	Use:   "multiclaude",
	Short: "Switch between Claude Code accounts with named profiles",
	Long: `multiclaude manages isolated profiles for Claude Code — each with its own
credentials and org context — and switches between them instantly.

Getting started:

  1. Log into a Claude Code account and save it as a profile:
       multiclaude add work

  2. Add another account:
       multiclaude add personal

  3. Switch between them any time:
       multiclaude use work
       multiclaude use personal`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	output.RegisterFlag(rootCmd)
	debug.RegisterFlag(rootCmd)
	config.RegisterFlag(rootCmd, "multiclaude")
	version.SetupFlag(rootCmd, versionInfo)

	rootCmd.AddCommand(version.NewCommand(versionInfo))
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(currentCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(renameCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(uninstallCmd)
}

// Execute runs the root command and exits on error.
func Execute() {
	err := rootCmd.Execute()
	errors.ExitWithError(err)
}
