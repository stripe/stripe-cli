package coopcmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/followups"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
)

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
	c := &coopAgentActionCmd{ensureSkill: ensureRepoStripeBestPracticesSkill}
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
			return runCoopStartFollowup(cmd, c.session, c.action, c.target, c.ensureSkill)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Parent session ID")
	c.cmd.Flags().StringVar(&c.action, "action", "", "Follow-up action ID")
	c.cmd.Flags().StringVar(&c.target, "target", "", "Detected deployment target")
	configureAgentCommand(c.cmd)
	return c
}

func runCoopNextAction(sessionID, completed string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return outputAgentError(fmt.Errorf("creating store: %w", err))
	}
	return runCoopNextActionWithStore(store, sessionID, completed)
}

// runCoopNextActionWithStore writes a still-waiting result to stdout with a
// zero exit code. Only genuine failures take the stderr/non-zero path: an agent
// that sees a non-zero exit treats the session as broken and gives up, which is
// exactly wrong when the developer is merely still deciding.
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

func runCoopStartFollowup(cmd *cobra.Command, parentSessionID, actionID, target string, ensureSkill func() error) error {
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
