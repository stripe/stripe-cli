package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	agentReportUsageEventName       = "agent.report_usage"
	agentReportUsageAcknowledgement = "Thank you for letting us know you used %s. If it was surprisingly good or bad, let us know via `stripe feedback --help`.\n"
)

type agentReportUsageCmd struct {
	cmd *cobra.Command

	skill string
}

type agentReportUsageEvent struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func newAgentReportUsageCmd() *agentReportUsageCmd {
	arc := &agentReportUsageCmd{}
	arc.cmd = &cobra.Command{
		Use:           "report_usage",
		Short:         "Report usage of agent tooling",
		Long:          "Report usage of agent tooling like Skills.",
		Args:          validators.NoArgs,
		RunE:          arc.runReportUsage,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	arc.cmd.Flags().StringVar(&arc.skill, "skill", "", "Name of the Stripe skill that was used (required)")

	return arc
}

func (arc *agentReportUsageCmd) runReportUsage(cmd *cobra.Command, _ []string) error {
	if arc.skill == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "--skill is required")
	}
	fmt.Fprintf(cmd.OutOrStdout(), agentReportUsageAcknowledgement, arc.skill)

	payload, err := json.Marshal(agentReportUsageEvent{
		Type: "skill",
		Name: arc.skill,
	})
	if err != nil {
		return nil
	}

	sendAgentEvent(commandContextOrBackground(cmd), agentReportUsageEventName, string(payload))
	return nil
}
