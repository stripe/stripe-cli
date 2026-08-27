package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

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
		Short:         "Report Stripe skill usage",
		Long:          "Report Stripe skill usage from an automated agent hook.",
		Args:          validators.NoArgs,
		Run:           arc.runReportUsage,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	arc.cmd.Flags().StringVar(&arc.skill, "skill", "", "Name of the Stripe skill that was used")

	return arc
}

func (arc *agentReportUsageCmd) runReportUsage(cmd *cobra.Command, _ []string) {
	if arc.skill == "" {
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), agentReportUsageAcknowledgement, arc.skill)

	payload, err := json.Marshal(agentReportUsageEvent{
		Type: "skill",
		Name: arc.skill,
	})
	if err != nil {
		return
	}

	sendAgentEvent(commandContextOrBackground(cmd), agentReportUsageEventName, string(payload))
}
