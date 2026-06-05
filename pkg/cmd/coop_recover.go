package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopRecoverCmd struct {
	cmd     *cobra.Command
	session string
	fix     bool
}

type recoverResponse struct {
	OK        bool   `json:"ok"`
	SessionID string `json:"session_id"`
	Diagnosis string `json:"diagnosis"`
	Action    string `json:"action,omitempty"`
	Next      string `json:"next,omitempty"`
}

func newCoopRecoverCmd() *coopRecoverCmd {
	rc := &coopRecoverCmd{}
	rc.cmd = &cobra.Command{
		Use:   "recover",
		Short: "Diagnose and fix a stuck co-op session",
		Long: `Inspects the current session for common issues and attempts to fix them.

Common issues detected:
- Step stuck in "active" with no agent polling (agent crashed)
- All steps done but session not marked complete
- No active steps and no pending steps (session is stuck)

Use --fix to automatically apply repairs. Without --fix, only reports the diagnosis.`,
		RunE: rc.runRecoverCmd,
	}

	rc.cmd.Flags().StringVar(&rc.session, "session", "", "Session ID (defaults to latest)")
	rc.cmd.Flags().BoolVar(&rc.fix, "fix", false, "Automatically fix detected issues")

	return rc
}

func (rc *coopRecoverCmd) runRecoverCmd(cmd *cobra.Command, args []string) error {
	store, err := coop.NewStore(Config.GetConfigFolder(""))
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var session *coop.Session
	if rc.session != "" {
		session, err = store.Read(rc.session)
	} else {
		session, err = store.LatestSession()
	}
	if err != nil {
		return outputCoopError("No session found.", "stripe coop run <blueprint>")
	}

	// Check: session already completed/aborted
	if session.Status != coop.SessionActive {
		return outputJSON(recoverResponse{
			OK:        true,
			SessionID: session.ID,
			Diagnosis: fmt.Sprintf("Session is already %s. No recovery needed.", session.Status),
			Next:      "stripe coop run <blueprint>",
		})
	}

	// Check: all steps done but session still active
	if session.IsComplete() {
		if rc.fix {
			session.Status = coop.SessionCompleted
			store.Write(session)
			return outputJSON(recoverResponse{
				OK:        true,
				SessionID: session.ID,
				Diagnosis: "All steps were done but session was still marked active.",
				Action:    "Marked session as completed.",
				Next:      "stripe coop next-steps",
			})
		}
		return outputJSON(recoverResponse{
			OK:        true,
			SessionID: session.ID,
			Diagnosis: "All steps are done but session is still marked active.",
			Next:      "stripe coop recover --fix",
		})
	}

	// Check: step stuck in active with no heartbeat
	activeNode, activeNum := session.ActiveNode()
	if activeNode != nil {
		age := store.HeartbeatAge(session.ID)
		agentPolling := age >= 0 && age < 5*1e9 // 5 seconds in nanoseconds

		if !agentPolling {
			if rc.fix {
				session.TransitionStep(activeNum, coop.StepReview)
				activeNode.Activity = ""
				store.Write(session)
				return outputJSON(recoverResponse{
					OK:        true,
					SessionID: session.ID,
					Diagnosis: fmt.Sprintf("Step %d (%s) was stuck in active with no agent polling.", activeNum, activeNode.Title),
					Action:    "Moved step to review. You can confirm it in the TUI (press c) or reject it (press r) to retry.",
					Next:      fmt.Sprintf("stripe coop join %s", session.ID),
				})
			}
			return outputJSON(recoverResponse{
				OK:        true,
				SessionID: session.ID,
				Diagnosis: fmt.Sprintf("Step %d (%s) appears stuck — no agent is polling. The agent may have crashed.", activeNum, activeNode.Title),
				Next:      "stripe coop recover --fix",
			})
		}

		// Agent is polling — nothing wrong
		return outputJSON(recoverResponse{
			OK:        true,
			SessionID: session.ID,
			Diagnosis: fmt.Sprintf("Step %d (%s) is active and the agent is polling. Session looks healthy.", activeNum, activeNode.Title),
		})
	}

	// Check: step stuck in review (agent not calling await)
	for i := range session.Chapters {
		for j := range session.Chapters[i].Nodes {
			if session.Chapters[i].Nodes[j].State == coop.StepReview {
				return outputJSON(recoverResponse{
					OK:        true,
					SessionID: session.ID,
					Diagnosis: fmt.Sprintf("Step %q is in review — waiting for your confirmation.", session.Chapters[i].Nodes[j].Title),
					Next:      fmt.Sprintf("stripe coop join %s", session.ID),
				})
			}
		}
	}

	// Check: no active, no review — find next pending
	nextStep := session.NextPendingStep(0)
	if nextStep > 0 {
		node, _ := session.NodeByNumber(nextStep)
		return outputJSON(recoverResponse{
			OK:        true,
			SessionID: session.ID,
			Diagnosis: fmt.Sprintf("No step is active. Next pending step is %d (%s). The agent needs to start it.", nextStep, node.Title),
			Next:      fmt.Sprintf("stripe coop step %d start --note=\"Resuming\"", nextStep),
		})
	}

	return outputJSON(recoverResponse{
		OK:        true,
		SessionID: session.ID,
		Diagnosis: "Session is in an unexpected state. Consider aborting and starting over.",
		Next:      "stripe coop stop --abort",
	})
}
