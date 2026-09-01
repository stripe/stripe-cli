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
	agentReportUsageAcknowledgement = "Thank you for letting us know you used %s %s. If it was surprisingly good or bad, let us know via `stripe feedback --help`.\n"
)

type agentReportUsageCmd struct {
	cmd *cobra.Command

	usageType string
	name      string
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
	arc.cmd.Flags().StringVar(&arc.usageType, "type", "", "Type of agent tooling that was used (required)")
	arc.cmd.Flags().StringVar(&arc.name, "name", "", "Name of the agent tooling that was used (required)")

	return arc
}

func (arc *agentReportUsageCmd) runReportUsage(cmd *cobra.Command, _ []string) error {
	if arc.usageType == "" && arc.name == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "--type and --name are required")
	}
	if arc.usageType == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "--type is required")
	}
	if arc.name == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "--name is required")
	}
	fmt.Fprintf(cmd.OutOrStdout(), agentReportUsageAcknowledgement, arc.usageType, arc.name)

	payload, err := json.Marshal(agentReportUsageEvent{
		Type: arc.usageType,
		Name: arc.name,
	})
	if err != nil {
		return nil
	}

	sendAgentEvent(commandContextOrBackground(cmd), agentReportUsageEventName, string(payload))
	return nil
}
