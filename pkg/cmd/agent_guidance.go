package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmd/agentguidance"
	"github.com/stripe/stripe-cli/pkg/config"
)

func newAgentGuidanceCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-guidance",
		Short: "Manage Stripe CLI agent guidance",
		Long: "Manage the agent guidance interstitial that helps AI agents " +
			"discover the right CLI surface for a task (public API vs. the " +
			"dynamic-API spec plugin).",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "snooze",
		Short: "Snooze the agent guidance message for the rest of today",
		RunE: func(c *cobra.Command, args []string) error {
			today := agentguidance.Today()
			if err := cfg.WriteConfigField(
				"agent_guidance.snoozed_until",
				agentguidance.SnoozeDate(today),
			); err != nil {
				return fmt.Errorf("failed to snooze agent guidance: %w", err)
			}
			fmt.Fprintln(c.OutOrStdout(), "✔ Agent guidance snoozed for the rest of today.")
			return nil
		},
	})

	return cmd
}
