package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/natikgadzhi/cli-kit/config"
	"github.com/natikgadzhi/cli-kit/table"
	"github.com/natikgadzhi/multiclaude/internal/claude"
	mcconfig "github.com/natikgadzhi/multiclaude/internal/config"
	"github.com/natikgadzhi/multiclaude/internal/keychain"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

// checkStatus represents the outcome of a single diagnostic check.
type checkStatus int

const (
	statusPass checkStatus = iota
	statusWarn
	statusFail
)

// checkResult holds one row of the doctor output table.
type checkResult struct {
	check      string
	status     checkStatus
	detail     string
	suggestion string
}

func (s checkStatus) String() string {
	switch s {
	case statusPass:
		return "pass"
	case statusWarn:
		return "warn"
	case statusFail:
		return "FAIL"
	default:
		return "?"
	}
}

func (s checkStatus) symbol() string {
	switch s {
	case statusPass:
		return "✓"
	case statusWarn:
		return "⚠"
	case statusFail:
		return "✗"
	default:
		return "?"
	}
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose common issues with profiles and credentials",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func runDoctor(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		configPath = config.DefaultPath("multiclaude")
	}
	cfg, err := mcconfig.Load(configPath)
	if err != nil {
		cfg = &mcconfig.Config{
			ClaudeHome: "~/.claude",
		}
	}

	// Expand tilde in ClaudeHome if still unexpanded.
	claudeHomePath := cfg.ClaudeHome
	if expanded, err := config.ExpandTilde(claudeHomePath); err == nil {
		claudeHomePath = expanded
	}

	ch := claude.NewClaudeHome(claudeHomePath)

	var results []checkResult

	// Check 1: Claude home exists.
	results = append(results, checkClaudeHome(ch))

	// Check 2: Config file valid.
	results = append(results, checkConfig(configPath))

	// Check 3: Profiles directory exists.
	profilesDir, profilesDirErr := mcconfig.ProfilesDir()
	results = append(results, checkProfilesDir(profilesDir, profilesDirErr))

	// Check 4: Per-profile checks (directory + keychain).
	if profilesDirErr == nil {
		store := profile.NewStore(profilesDir, ch)
		profiles, err := store.List()
		if err == nil {
			for _, p := range profiles {
				results = append(results, checkProfileDir(store, p.Name))
				results = append(results, checkProfileKeychain(p.Name))
			}
		}
	}

	// Check 5: Active profile consistency.
	if profilesDirErr == nil {
		store := profile.NewStore(profilesDir, ch)
		if activeFile, err := mcconfig.ActiveFilePath(); err == nil {
			store.SetActiveFile(activeFile)
		}
		results = append(results, checkActiveProfile(ch, store))
	}

	// Check 6: Claude Code CLI available.
	results = append(results, checkClaudeCLI())

	// Render the table.
	t := table.New()
	t.Header("Check", "Status", "Detail")
	for _, r := range results {
		t.Row(r.check, r.status.symbol(), r.detail)
	}
	if err := t.Flush(); err != nil {
		return err
	}

	// Print suggestions for failures.
	hasFail := false
	for _, r := range results {
		if r.status == statusFail {
			hasFail = true
			if r.suggestion != "" {
				fmt.Fprintf(os.Stderr, "\n  %s: %s\n", r.check, r.suggestion)
			}
		}
	}

	if hasFail {
		return fmt.Errorf("one or more checks failed")
	}
	return nil
}

func checkClaudeHome(ch *claude.ClaudeHome) checkResult {
	if ch.Exists() {
		return checkResult{
			check:  "Claude home exists",
			status: statusPass,
			detail: ch.Path,
		}
	}
	return checkResult{
		check:      "Claude home exists",
		status:     statusFail,
		detail:     ch.Path + " not found",
		suggestion: "Run Claude Code once to create ~/.claude, or check claude_home in config.toml",
	}
}

func checkConfig(path string) checkResult {
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		expanded = path
	}

	if _, err := os.Stat(expanded); os.IsNotExist(err) {
		return checkResult{
			check:  "Config file",
			status: statusPass,
			detail: "not found, using defaults",
		}
	}

	_, err = mcconfig.Load(path)
	if err != nil {
		return checkResult{
			check:  "Config file",
			status: statusFail,
			detail: fmt.Sprintf("invalid: %v", err),
			suggestion: "Fix the TOML syntax in " + expanded + ". Minimal example:\n" +
				"  default_profile = \"work\"\n" +
				"  claude_home = \"~/.claude\"",
		}
	}
	return checkResult{
		check:  "Config file",
		status: statusPass,
		detail: expanded,
	}
}

func checkProfilesDir(dir string, err error) checkResult {
	if err != nil {
		return checkResult{
			check:      "Profiles directory",
			status:     statusFail,
			detail:     fmt.Sprintf("error: %v", err),
			suggestion: "Check filesystem permissions for ~/.config/multiclaude/profiles",
		}
	}
	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		return checkResult{
			check:  "Profiles directory",
			status: statusWarn,
			detail: "not created yet (no profiles saved)",
		}
	}
	return checkResult{
		check:  "Profiles directory",
		status: statusPass,
		detail: dir,
	}
}

func checkProfileDir(store *profile.Store, name string) checkResult {
	dir := store.ProfileDir(name)
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return checkResult{
			check:      fmt.Sprintf("Profile %q directory", name),
			status:     statusFail,
			detail:     "missing or not a directory",
			suggestion: fmt.Sprintf("Re-create the profile with: multiclaude add %s", name),
		}
	}
	return checkResult{
		check:  fmt.Sprintf("Profile %q directory", name),
		status: statusPass,
		detail: dir,
	}
}

func checkProfileKeychain(name string) checkResult {
	if keychain.HasCredentials(name) {
		return checkResult{
			check:  fmt.Sprintf("Profile %q keychain", name),
			status: statusPass,
			detail: "credentials present",
		}
	}
	return checkResult{
		check:      fmt.Sprintf("Profile %q keychain", name),
		status:     statusFail,
		detail:     "no keychain entry found",
		suggestion: fmt.Sprintf("Re-add credentials: multiclaude remove %s && multiclaude add %s", name, name),
	}
}

func checkActiveProfile(ch *claude.ClaudeHome, store *profile.Store) checkResult {
	name, err := store.ActiveProfileName()
	if err != nil {
		return checkResult{
			check:      "Active profile",
			status:     statusFail,
			detail:     fmt.Sprintf("error reading active profile: %v", err),
			suggestion: "Switch to a profile to fix state: multiclaude use <name>",
		}
	}

	if name == "" {
		if !ch.Exists() {
			return checkResult{
				check:      "Active profile",
				status:     statusWarn,
				detail:     "no Claude home directory and no active profile",
				suggestion: "Log into Claude Code first, then run: multiclaude add <name>",
			}
		}
		return checkResult{
			check:  "Active profile",
			status: statusWarn,
			detail: "Claude home exists but no profile is active",
		}
	}

	if !store.Exists(name) {
		return checkResult{
			check:      "Active profile",
			status:     statusFail,
			detail:     fmt.Sprintf("active file references missing profile %q", name),
			suggestion: "Switch to a valid profile: multiclaude use <name>",
		}
	}

	return checkResult{
		check:  "Active profile",
		status: statusPass,
		detail: fmt.Sprintf("profile %q", name),
	}
}

func checkClaudeCLI() checkResult {
	path, err := exec.LookPath("claude")
	if err != nil {
		return checkResult{
			check:      "Claude CLI in PATH",
			status:     statusWarn,
			detail:     "claude command not found",
			suggestion: "Install Claude Code: https://docs.anthropic.com/en/docs/claude-code",
		}
	}
	return checkResult{
		check:  "Claude CLI in PATH",
		status: statusPass,
		detail: path,
	}
}
