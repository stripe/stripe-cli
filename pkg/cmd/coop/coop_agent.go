package coopcmd

import (
	"errors"
	"fmt"

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
	cmd         *cobra.Command
	ensureSkill func() error
	session     string
	step        int
	note        string

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
	}
	ac.cmd.AddCommand(newCoopAgentStartWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportWorkCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentReportCheckCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentSkipCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentAwaitReviewCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentNextActionCmd().cmd)
	ac.cmd.AddCommand(newCoopAgentStartFollowupCmd().cmd)
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
				return outputAgentError(err)
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
				return outputAgentError(err)
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
				return outputAgentError(err)
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
				return outputAgentError(err)
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

func newCoopAgentStartFollowupCmd() *coopAgentActionCmd {
	c := &coopAgentActionCmd{ensureSkill: ensureRepoStripeBestPracticesSkill}
	c.cmd = &cobra.Command{
		Use:   "start-followup",
		Short: "Start an internal guided follow-up session",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCoopStartFollowup(cmd, c.session, c.action, c.target, c.ensureSkill)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Parent session ID")
	c.cmd.Flags().StringVar(&c.action, "action", "", "Follow-up action ID")
	c.cmd.Flags().StringVar(&c.target, "target", "", "Detected deployment target")
	mustMarkFlagRequired(c.cmd, "session")
	mustMarkFlagRequired(c.cmd, "action")
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
	return runCoopNextActionWithStore(store, sessionID, completed)
}

func runCoopNextActionWithStore(store helpers.Store, sessionID, completed string) error {
	resp, err := helpers.Run(store, helpers.Input{SessionID: sessionID, Completed: completed})
	if errors.Is(err, helpers.ErrNoSession) {
		return outputCoopError("No session found.", "stripe coop run <blueprint>")
	}
	if err != nil {
		return outputCoopError(err.Error(), nextActionHint(sessionID))
	}
	return outputJSON(resp)
}

func nextActionHint(sessionID string) string {
	if sessionID == "" {
		return "stripe coop agent next-action --session=<session>"
	}
	return fmt.Sprintf("stripe coop agent next-action --session=%s", sessionID)
}

func runCoopStartFollowup(cmd *cobra.Command, parentSessionID, actionID, target string, ensureSkill func() error) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	parent, err := store.Read(parentSessionID)
	if err != nil {
		return outputCoopError(fmt.Sprintf("Parent session %q not found.", parentSessionID), "stripe coop agent next-action --session=<session>")
	}

	action, err := followups.GuidedActionByID(actionID, target)
	if err != nil {
		return outputCoopError(err.Error(), "stripe coop agent start-followup --session=<session> --action=deploy")
	}
	if err := validateFollowupParent(parent, action.ID); err != nil {
		return outputCoopError(err.Error(), "stripe coop agent next-action --session="+parent.ID)
	}
	if ensureSkill != nil {
		if err := ensureSkill(); err != nil {
			warnRepoStripeBestPracticesSkill(cmd, err)
		}
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
		return fmt.Errorf("writing guided follow-up session: %w", err)
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
		// Emit a structured ok:false response (on stdout, like every other agent
		// command) so an agent parsing JSON always gets an error + recovery hint,
		// even on infra failures (e.g. a heartbeat/store write error mid-await).
		resp = coop.CommandResponse{
			OK:    false,
			Error: err.Error(),
			Hint:  "stripe coop status",
			Next:  "stripe coop status",
		}
	}
	if outErr := outputJSON(resp); outErr != nil {
		return outErr
	}
	if !resp.OK {
		return RenderedError{}
	}
	return nil
}
