package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	agentReportUsageEventName = "skill_used"
	agentReportUsageMessage   = "Thank you for reporting. If anything about the tool was surprisingly good or bad, Stripe welcomes feedback from agents via `stripe feedback --help`"
)

type agentReportUsageCmd struct {
	cmd *cobra.Command

	skill  string
	debug  bool
	dryRun bool
}

type agentReportUsageContext struct {
	SkillName string `json:"skillName"`
}

type agentReportUsageResponse struct {
	Message string `json:"message"`
}

func newAgentReportUsageCmd() *agentReportUsageCmd {
	arc := &agentReportUsageCmd{}
	arc.cmd = &cobra.Command{
		Use:           "report_usage",
		Args:          validators.NoArgs,
		Hidden:        true,
		RunE:          arc.runReportUsage,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	arc.cmd.Flags().StringVar(&arc.skill, "skill", "", "Skill name to report")
	arc.cmd.Flags().BoolVar(&arc.debug, "debug", false, "Log arguments and telemetry calls")
	arc.cmd.Flags().BoolVar(&arc.dryRun, "dry-run", false, "Log the telemetry event without emitting it")
	_ = arc.cmd.MarkFlagRequired("skill")

	return arc
}

func (arc *agentReportUsageCmd) runReportUsage(cmd *cobra.Command, _ []string) error {
	if arc.debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: report_usage arguments: skill=%q dry-run=%t\n", arc.skill, arc.dryRun)
	}
	if arc.skill == "" {
		return errorcategory.New(errorcategory.UserInput, "--skill must not be empty")
	}

	additionalContext, err := json.Marshal(agentReportUsageContext{SkillName: arc.skill})
	if err != nil {
		return fmt.Errorf("encoding skill usage context: %w", err)
	}

	if arc.dryRun {
		fmt.Fprintf(cmd.ErrOrStderr(), "Dry run: would emit telemetry event %q with additional context %s\n", agentReportUsageEventName, additionalContext)
	} else if telemetryClient := stripe.GetTelemetryClient(commandContextOrBackground(cmd)); telemetryClient != nil {
		if arc.debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: emitting telemetry event %q with additional context %s\n", agentReportUsageEventName, additionalContext)
		}
		telemetryClient.SendEvent(commandContextOrBackground(cmd), agentReportUsageEventName, string(additionalContext))
	} else if arc.debug {
		fmt.Fprintln(cmd.ErrOrStderr(), "Debug: telemetry client unavailable; event was not emitted")
	}

	if arc.debug {
		fmt.Fprintln(cmd.ErrOrStderr(), "Debug: writing JSON response")
	}
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(agentReportUsageResponse{Message: agentReportUsageMessage}); err != nil {
		return fmt.Errorf("writing report usage response: %w", err)
	}
	return nil
}
