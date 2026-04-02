package cmd

import (
	"fmt"

	"github.com/natikgadzhi/cli-kit/output"
	"github.com/natikgadzhi/cli-kit/table"
	"github.com/natikgadzhi/multiclaude/internal/profile"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all profiles",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

// profileListJSON is the JSON representation of a profile for list output.
type profileListJSON struct {
	Name             string `json:"name"`
	Email            string `json:"email,omitempty"`
	SubscriptionType string `json:"subscription_type,omitempty"`
	Status           string `json:"status"`
}

// profileListRenderer implements output.TableRenderer for a slice of profiles.
type profileListRenderer struct {
	profiles []profile.Profile
}

func (r *profileListRenderer) RenderTable(t *table.Table) {
	t.RowBorders = true
	t.Header("Profile", "Email", "Status")
	for _, p := range r.profiles {
		status := ""
		if p.IsActive {
			status = "active"
		}
		email := p.Email
		if p.SubscriptionType != "" {
			email = fmt.Sprintf("%s (%s)", p.Email, p.SubscriptionType)
		}
		t.Row(p.Name, email, status)
	}
}

func runList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig(cmd)
	if err != nil {
		return err
	}

	store, err := newProfileStore(cfg)
	if err != nil {
		return err
	}

	profiles, err := store.List()
	if err != nil {
		return fmt.Errorf("listing profiles: %w", err)
	}

	format := output.Resolve(cmd)

	if len(profiles) == 0 {
		if output.IsTable(format) {
			fmt.Fprintln(cmd.OutOrStdout(), "No profiles found. Run 'multiclaude add <name>' to create one.")
		} else {
			// JSON: empty array
			output.PrintJSON([]any{})
		}
		return nil
	}

	// Build JSON data.
	jsonData := make([]profileListJSON, len(profiles))
	for i, p := range profiles {
		status := ""
		if p.IsActive {
			status = "active"
		}
		jsonData[i] = profileListJSON{
			Name:             p.Name,
			Email:            p.Email,
			SubscriptionType: p.SubscriptionType,
			Status:           status,
		}
	}

	renderer := &profileListRenderer{profiles: profiles}
	return output.Print(format, jsonData, renderer)
}
