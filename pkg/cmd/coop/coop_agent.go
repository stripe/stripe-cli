package coopcmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/followups"
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
	ac.cmd.AddCommand(newCoopAgentNextActionCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentStartFollowupCmd().cmd)
	configureAgentCommand(ac.cmd)
	return ac
}

func newCoopAgentStartWorkCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "start-work",
		Short: "Mark a node as active",
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
			service, err := newWorkflowService()
			if err != nil {
				return outputAgentError(err)
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
			resp, err := service.ReportCheck(c.session, c.step, c.check, c.passed)
			return outputAgentResponse(resp, err)
		},
	}
	c.addSessionStepFlags()
	c.cmd.Flags().StringVar(&c.check, "check", "", "Verification check label")
	c.cmd.Flags().BoolVar(&c.passed, "passed", false, "Whether the verification passed")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentSkipCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "skip",
		Short: "Skip a node",
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

func newCoopAgentNextActionCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "next-action",
		Short: "Wait for or record the developer's next action",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(c.session) == "" {
				return outputCoopError(
					"--session flag is required",
					"Retry next-action with the intended session.",
					coop.NextActionTemplate(),
				)
			}
			return runCoopNextAction(c.session, c.completed)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	c.cmd.Flags().StringVar(&c.completed, "completed", "", "Mark a next action as completed")
	configureAgentCommand(c.cmd)
	return c
}

func newCoopAgentStartFollowupCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{}
	c.cmd = &cobra.Command{
		Use:   "start-followup",
		Short: "Start an internal guided follow-up session",
		Args:  agentNoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(c.session) == "" || strings.TrimSpace(c.action) == "" {
				return outputCoopError(
					"--session and --action flags are required",
					"Provide the parent session and an offered follow-up action.",
					coop.StartFollowupTemplate(""),
				)
			}
			return runCoopStartFollowup(c.session, c.action, c.target)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Parent session ID")
	c.cmd.Flags().StringVar(&c.action, "action", "", "Follow-up action ID")
	c.cmd.Flags().StringVar(&c.target, "target", "", "Detected deployment target")
	configureAgentCommand(c.cmd)
	return c
}

func (c *coopAgentActionCmd) addSessionStepFlags() {
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID")
	c.cmd.Flags().IntVar(&c.step, "step", 0, "1-based node number")
}

func (c *coopAgentActionCmd) validateSessionStep(action string) error {
	if strings.TrimSpace(c.session) != "" && c.step > 0 {
		return nil
	}
	return outputCoopError(
		"--session and a positive --step are required",
		"Provide the Co-op session ID and 1-based node number.",
		coop.SessionStepTemplate(action),
	)
}

func configureAgentCommand(cmd *cobra.Command) {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return outputCoopError(
			err.Error(),
			"Correct the command flags and retry.",
			coop.Continue(coop.StatusCommand("")),
		)
	})
}

func agentNoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}
	return outputCoopError(
		fmt.Sprintf("%s does not accept positional arguments", cmd.CommandPath()),
		"Remove the unexpected positional arguments and retry.",
		coop.Continue(coop.StatusCommand("")),
	)
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
		return outputAgentError(fmt.Errorf("creating store: %w", err))
	}
	return runCoopNextActionWithStore(store, sessionID, completed)
}

func runCoopNextActionWithStore(store helpers.Store, sessionID, completed string) error {
	resp, err := helpers.Run(store, helpers.Input{SessionID: sessionID, Completed: completed})
	if errors.Is(err, helpers.ErrNoSession) {
		return outputCoopError(
			"No session found.",
			"Start a Co-op session before requesting a next action.",
			coop.RunTemplate(),
		)
	}
	if err != nil {
		return outputCoopError(
			err.Error(),
			"Retry the next-action wait.",
			coop.Continue(coop.NextActionCommand(sessionID, "")),
		)
	}
	return outputJSON(resp)
}

func runCoopStartFollowup(parentSessionID, actionID, target string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return outputAgentError(fmt.Errorf("creating store: %w", err))
	}

	parent, err := store.Read(parentSessionID)
	if err != nil {
		return outputCoopError(
			fmt.Sprintf("Parent session %q not found.", parentSessionID),
			"Inspect active and completed Co-op sessions.",
			coop.Continue(coop.StatusCommand("")),
		)
	}

	action, err := followups.GuidedActionByID(actionID, target)
	if err != nil {
		return outputCoopError(
			err.Error(),
			"Use an action offered by the parent session.",
			coop.StartFollowupTemplate(parentSessionID),
		)
	}
	if err := validateFollowupParent(parent, action.ID); err != nil {
		return outputCoopError(
			err.Error(),
			"Return to the parent session's next-action selection.",
			coop.Continue(coop.NextActionCommand(parent.ID, "")),
		)
	}

	settings := make(map[string]string, len(parent.Settings)+1)
	for key, value := range parent.Settings {
		settings[key] = value
	}
	if target != "" {
		settings["deploy_target"] = target
	}

	sessionID := "coop_" + generateShortID()
	session := coop.NewSessionFromGuidedAction(action, sessionID, coop.GuidedActionSessionOptions{
		ParentSessionID: parent.ID,
		ParentStepID:    action.ID,
		Settings:        settings,
		UsedSandbox:     parent.UsedSandbox || coopSandboxClaimURL() != "",
	})
	if err := store.Write(session); err != nil {
		return outputAgentError(fmt.Errorf("writing guided follow-up session: %w", err))
	}

	return outputJSON(newCoopAgentGuidedActionResponse(action, session))
}

func validateFollowupParent(parent *coop.Session, actionID string) error {
	if parent.Status != coop.SessionCompleted {
		return fmt.Errorf("parent session %q is not completed", parent.ID)
	}
	if parent.NextSteps == nil {
		return fmt.Errorf("parent session %q has no next-step suggestions", parent.ID)
	}
	for _, completed := range parent.NextSteps.Completed {
		if completed == actionID {
			return fmt.Errorf("follow-up action %q is already completed for parent session %q", actionID, parent.ID)
		}
	}
	for _, suggestion := range parent.NextSteps.Suggestions {
		if suggestion.ID == actionID {
			return nil
		}
	}
	return fmt.Errorf("follow-up action %q is not available for parent session %q", actionID, parent.ID)
}

// outputAgentError renders err as a structured agent JSON response. Used for
// failures that happen before a workflow CommandResponse exists (e.g.
// newWorkflowService / store creation), so agent commands never emit a bare
// plain-text error on that path.
func outputAgentError(err error) error {
	return outputAgentResponse(coop.CommandResponse{}, err)
}

func outputAgentResponse(resp coop.CommandResponse, err error) error {
	if err != nil {
		resp = protocolFailure(err.Error())
	}
	if validationErr := resp.Validate(); validationErr != nil {
		resp = protocolFailure("invalid Co-op protocol response: " + validationErr.Error())
	}
	if !resp.OK {
		if resp.Recovery == nil {
			resp.Recovery = defaultAgentRecovery()
		}
		if outErr := outputJSONTo(os.Stderr, resp); outErr != nil {
			return outErr
		}
		return RenderedError{}
	}
	return outputJSON(resp)
}

func protocolFailure(message string) coop.CommandResponse {
	return coop.CommandResponse{
		OK:       false,
		Error:    message,
		Recovery: defaultAgentRecovery(),
	}
}

func defaultAgentRecovery() *coop.Recovery {
	return coop.Continue(coop.StatusCommand("")).
		Recovery("Inspect the current Co-op session before retrying.")
}
