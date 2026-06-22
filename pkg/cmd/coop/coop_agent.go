package coopcmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

type coopAgentCmd struct {
	cmd *cobra.Command
}

type coopAgentActionCmd struct {
	cmd     *cobra.Command
	session string
	step    int
	note    string

	file    string
	lines   string
	snippet string
	check   string
	passed  bool

	completed string
}

func newCoopAgentCmd() *coopAgentCmd {
	ac := &coopAgentCmd{}
	ac.cmd = &cobra.Command{
		Use:   "agent",
		Short: "Agent-facing co-op lifecycle commands",
		Long:  "Typed commands used by agents to report co-op progress and wait for human review.",
	}
	ac.cmd.AddCommand(newCoopAgentStartWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportCheckCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentSkipCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentAwaitReviewCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentNextActionCmd().cmd)
	return ac
}

func newCoopAgentStartWorkCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "start-work",
		Short: "Mark a node as active",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newWorkflowService()
			if err != nil {
				return err
			}
			resp, err := service.StartWork(c.session, c.step, c.note)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.note, "note", "", "Activity note")
	return c
}

func newCoopAgentReportWorkCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "report-work",
		Short: "Report completed implementation work",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newWorkflowService()
			if err != nil {
				return err
			}
			resp, err := service.ReportWork(c.session, c.step, workflow.ReportWorkInput{
				File:    c.file,
				Lines:   c.lines,
				Snippet: c.snippet,
				Note:    c.note,
			}, false)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.file, "file", "", "File path for implementation")
	c.cmd.Flags().StringVar(&c.lines, "lines", "", "Line range, e.g. 1-15")
	c.cmd.Flags().StringVar(&c.snippet, "snippet", "", "Code snippet")
	c.cmd.Flags().StringVar(&c.note, "note", "", "Implementation summary")
	return c
}

func newCoopAgentReportCheckCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "report-check",
		Short: "Report a verification check",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newWorkflowService()
			if err != nil {
				return err
			}
			resp, err := service.ReportCheck(c.session, c.step, c.check, c.passed)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.check, "check", "", "Verification check label")
	c.cmd.Flags().BoolVar(&c.passed, "passed", false, "Whether the verification passed")
	return c
}

func newCoopAgentSkipCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "skip",
		Short: "Skip a node",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newWorkflowService()
			if err != nil {
				return err
			}
			resp, err := service.Skip(c.session, c.step, c.note)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.note, "note", "", "Skip reason")
	return c
}

func newCoopAgentAwaitReviewCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "await-review",
		Short: "Block until the developer confirms or requests changes",
		RunE: func(cmd *cobra.Command, args []string) error {
			service, err := newWorkflowService()
			if err != nil {
				return err
			}
			resp, err := service.AwaitReview(c.session, c.step)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	return c
}

func newCoopAgentNextActionCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "next-action",
		Short: "Wait for or record the developer's next action",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCoopNextAction(c.session, c.completed)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	c.cmd.Flags().StringVar(&c.completed, "completed", "", "Mark a next action as completed")
	mustMarkFlagRequired(c.cmd, "session")
	return c
}

func (c *coopAgentActionCmd) addSessionStepFlags() {
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	c.cmd.Flags().IntVar(&c.step, "step", 0, "1-based node number")
	mustMarkFlagRequired(c.cmd, "session")
	mustMarkFlagRequired(c.cmd, "step")
}

func newWorkflowService() (*workflow.Service, error) {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return nil, fmt.Errorf("creating store: %w", err)
	}
	return workflow.NewService(store), nil
}

func runCoopNextAction(sessionID, completed string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}
	resp, err := helpers.Run(store, helpers.Input{SessionID: sessionID, Completed: completed})
	if errors.Is(err, helpers.ErrNoSession) {
		return outputCoopError("No session found.", "stripe coop run <blueprint>")
	}
	if err != nil {
		return err
	}
	return outputJSON(resp)
}

func outputAgentResponse(resp coop.CommandResponse, err error) error {
	if err != nil {
		return err
	}
	if outErr := outputJSON(resp); outErr != nil {
		return outErr
	}
	if !resp.OK {
		return RenderedError{}
	}
	return nil
}
