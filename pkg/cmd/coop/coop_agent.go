package coopcmd

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

type coopAgentCmd struct {
	cmd *cobra.Command
}

type coopAgentActionCmd struct {
	cmd         *cobra.Command
	ensureSkill func() error
	session     string
	step        int
	note        string

	file    string
	lines   string
	snippet string
	check   string
	detail  string
	passed  bool
	outputs []string

	completed string
	action    string
	target    string
}

func newCoopAgentCmd() *coopAgentCmd {
	ac := &coopAgentCmd{}
	ac.cmd = &cobra.Command{
		Use:   "agent",
		Short: "Agent-facing co-op lifecycle commands",
		Long:  "Typed commands used by agents to report co-op progress and wait for human review.",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return outputCoopError(
				"stripe coop agent requires an action",
				"Run the agent lifecycle action returned by the previous Co-op response.",
				coop.Continue(coop.StatusCommand("")),
			)
		},
	}
	ac.cmd.AddCommand(newCoopAgentStartWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportCheckCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentSkipCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentAwaitReviewCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentResumeCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentNextActionCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentStartFollowupCmd().cmd)
	configureAgentCommand(ac.cmd)
	return ac
}

func newCoopAgentStartWorkCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "start-work",
		Short: "Mark a task as active",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validateSessionStep("start-work"); err != nil {
				return err
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.StartWork(c.session, c.step, c.note)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.note, "note", "", "Activity note")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentReportWorkCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "report-work",
		Short: "Report completed implementation work",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validateSessionStep("report-work"); err != nil {
				return err
			}
			outputs, err := parseReportedOutputs(c.outputs)
			if err != nil {
				return outputCoopError(
					err.Error(),
					"Use --output field=value or --output source:field=value.",
					coop.ReportWorkOutputTemplate(c.session, c.step),
				)
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.ReportWork(c.session, c.step, workflow.ReportWorkInput{
				File:    c.file,
				Lines:   c.lines,
				Snippet: c.snippet,
				Note:    c.note,
				Outputs: outputs,
			}, false)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.file, "file", "", "File path for implementation")
	c.cmd.Flags().StringVar(&c.lines, "lines", "", "Line range, e.g. 1-15")
	c.cmd.Flags().StringVar(&c.snippet, "snippet", "", "Code snippet")
	c.cmd.Flags().StringVar(&c.note, "note", "", "Implementation summary")
	c.cmd.Flags().StringArrayVar(&c.outputs, "output", nil, "Produced value as field=value or source:field=value (repeatable)")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentReportCheckCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "report-check",
		Short: "Report a verification check",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validateSessionStep("report-check"); err != nil {
				return err
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.ReportCheck(c.session, c.step, c.check, c.detail, c.passed)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.check, "check", "", "Short label for what was checked, e.g. \"Webhook signature verified\". Keep it to one line — it is what the reviewer sees")
	c.cmd.Flags().StringVar(&c.detail, "detail", "", "Full output or reasoning behind the result. Shown only when the reviewer asks for it, so command logs and long explanations belong here rather than in --check")
	c.cmd.Flags().BoolVar(&c.passed, "passed", false, "Whether the verification passed")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentSkipCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "skip",
		Short: "Skip a task",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validateSessionStep("skip"); err != nil {
				return err
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.Skip(c.session, c.step, c.note)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.note, "note", "", "Skip reason")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentAwaitReviewCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "await-review",
		Short: "Block until the developer confirms or requests changes",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.validateSessionStep("await-review"); err != nil {
				return err
			}
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
			}
			resp, err := service.AwaitReview(c.session, c.step)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	configureAgentCommand(c.cmd)
	return c
}
