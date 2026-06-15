package coopcmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

type coopDebugAgentCmd struct {
	cmd     *cobra.Command
	session string
	delay   time.Duration
}

func newCoopDebugAgentCmd() *coopDebugAgentCmd {
	dc := &coopDebugAgentCmd{delay: 650 * time.Millisecond}
	dc.cmd = &cobra.Command{
		Use:    "debug-agent",
		Short:  "Run a deterministic fake co-op agent",
		Hidden: true,
		RunE:   dc.runDebugAgentCmd,
	}

	dc.cmd.Flags().StringVar(&dc.session, "session", "", "Session ID to drive")
	dc.cmd.Flags().DurationVar(&dc.delay, "delay", dc.delay, "Delay between active and review states")
	mustMarkFlagHidden(dc.cmd, "delay")

	return dc
}

func (dc *coopDebugAgentCmd) runDebugAgentCmd(cmd *cobra.Command, args []string) error {
	if dc.session == "" {
		return fmt.Errorf("--session is required")
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	agent := &coopDebugAgent{
		store:                    store,
		sessionID:                dc.session,
		delay:                    dc.delay,
		pollInterval:             500 * time.Millisecond,
		out:                      os.Stdout,
		waitForNextStepSelection: true,
	}
	return agent.run(cmd.Context())
}

type coopDebugAgent struct {
	store                    *coop.Store
	sessionID                string
	delay                    time.Duration
	pollInterval             time.Duration
	out                      io.Writer
	waitForNextStepSelection bool
}

func (a *coopDebugAgent) run(ctx context.Context) error {
	a.logf("debug agent attached to %s", a.sessionID)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		session, err := a.store.Read(a.sessionID)
		if err != nil {
			return err
		}

		if session.IsComplete() {
			return a.completeSession(ctx, session)
		}

		if step := firstStepWithState(session, coop.NodeActive); step > 0 {
			if err := a.completeActiveStep(ctx, step); err != nil {
				return err
			}
			continue
		}

		if step := firstStepWithState(session, coop.NodeReview); step > 0 {
			if shouldContinueStepBeforeReview(session, step) {
				sessionStep, stepIndex, _, err := session.StepByNodeNumber(step)
				if err != nil {
					return err
				}
				if next := helpers.NextPendingNodeInStep(session, stepIndex, step); next > 0 {
					a.logf("step %q still has pending work; continuing with node %d", sessionStep.Title, next)
					if err := a.startStep(next); err != nil {
						return err
					}
					continue
				}
			}
			if err := a.awaitReview(ctx, step); err != nil {
				return err
			}
			continue
		}

		if step := firstStepWithState(session, coop.NodePending); step > 0 {
			if err := a.startStep(step); err != nil {
				return err
			}
			continue
		}

		if err := a.sleep(ctx, a.pollInterval); err != nil {
			return err
		}
	}
}

func (a *coopDebugAgent) startStep(step int) error {
	session, err := a.store.Read(a.sessionID)
	if err != nil {
		return err
	}
	node, err := session.NodeByNumber(step)
	if err != nil {
		return err
	}
	if node.State != coop.NodePending {
		return nil
	}

	resp, err := workflow.NewService(a.store).StartWork(a.sessionID, step, "Debug agent working: "+node.Title)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	a.logf("step %d active: %s", step, node.Title)
	return nil
}

func (a *coopDebugAgent) completeActiveStep(ctx context.Context, step int) error {
	if err := a.sleep(ctx, a.delay); err != nil {
		return err
	}

	session, err := a.store.Read(a.sessionID)
	if err != nil {
		return err
	}
	node, err := session.NodeByNumber(step)
	if err != nil {
		return err
	}
	if node.State != coop.NodeActive {
		return nil
	}

	service := workflow.NewService(a.store)
	resp, err := service.ReportCheck(a.sessionID, step, "Debug agent deterministic check", true)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	resp, err = service.ReportWork(a.sessionID, step, workflow.ReportWorkInput{
		File:  "debug/" + safeDebugFileName(node.Key) + ".txt",
		Lines: "1-1",
		Note:  "Deterministic debug agent completed " + node.Title,
	}, false)
	if err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	a.logf("step %d %s: %s", step, resp.State, node.Title)
	return nil
}

func (a *coopDebugAgent) awaitReview(ctx context.Context, step int) error {
	session, err := a.store.Read(a.sessionID)
	if err != nil {
		return err
	}
	node, err := session.NodeByNumber(step)
	if err != nil {
		return err
	}

	if helpers.StepReviewApplies(session, step) {
		step, stepIndex, _, err := session.StepByNodeNumber(step)
		if err != nil {
			return err
		}
		a.logf("waiting for step review: %s", step.Title)
		return a.awaitStepReview(ctx, stepIndex)
	}

	a.logf("waiting for review: step %d %s", step, node.Title)
	for {
		if err := a.store.WriteHeartbeat(a.sessionID); err != nil {
			return err
		}
		if err := a.sleep(ctx, a.pollInterval); err != nil {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			return err
		}

		session, err = a.store.Read(a.sessionID)
		if err != nil {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		node, err = session.NodeByNumber(step)
		if err != nil {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		if node.State != coop.NodeReview {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			a.logf("review released: step %d is %s", step, node.State)
			return nil
		}
	}
}

func (a *coopDebugAgent) awaitStepReview(ctx context.Context, stepIndex int) error {
	for {
		if err := a.store.WriteHeartbeat(a.sessionID); err != nil {
			return err
		}
		if err := a.sleep(ctx, a.pollInterval); err != nil {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			return err
		}

		session, err := a.store.Read(a.sessionID)
		if err != nil {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		if active := session.FirstActiveNodeInStep(stepIndex); active > 0 {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			a.logf("step requested changes; rerunning from step %d", active)
			return nil
		}
		if !session.StepHasReview(stepIndex) {
			_ = a.store.RemoveHeartbeat(a.sessionID)
			a.logf("step review released")
			return nil
		}
	}
}

func (a *coopDebugAgent) completeSession(ctx context.Context, session *coop.Session) error {
	if session.Status != coop.SessionCompleted || session.NextSteps == nil || len(session.NextSteps.Suggestions) == 0 {
		suggestions := helpers.BuildSuggestions(session, helpers.DetectProjectEnvironment())
		a.logf("all steps complete; showing next steps")
		if err := helpers.ShowSuggestions(a.store, session, suggestions, ""); err != nil {
			if isVersionConflict(err) {
				return nil
			}
			return err
		}
	}

	if !a.waitForNextStepSelection {
		return nil
	}

	for {
		if err := a.sleep(ctx, a.pollInterval); err != nil {
			return err
		}

		session, err := a.store.Read(a.sessionID)
		if err != nil {
			return err
		}
		if session.NextSteps == nil || session.NextSteps.Selected == "" {
			continue
		}

		selected := session.NextSteps.Selected
		session.NextSteps.Selected = ""
		if err := a.store.Write(session); err != nil && !isVersionConflict(err) {
			return err
		}
		a.logf("next step selected: %s", selected)
		return nil
	}
}

func firstStepWithState(session *coop.Session, state coop.NodeState) int {
	step := 0
	for i := range session.Steps {
		for j := range session.Steps[i].Nodes {
			step++
			if session.Steps[i].Nodes[j].State == state {
				return step
			}
		}
	}
	return 0
}

func shouldContinueStepBeforeReview(session *coop.Session, step int) bool {
	if !helpers.StepReviewApplies(session, step) {
		return false
	}
	_, stepIndex, _, err := session.StepByNodeNumber(step)
	if err != nil {
		return false
	}
	return !session.StepReadyForReview(stepIndex)
}

func safeDebugFileName(key string) string {
	if key == "" {
		return "step"
	}
	return strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(key)
}

func isVersionConflict(err error) bool {
	return err != nil && strings.Contains(err.Error(), "version conflict")
}

func (a *coopDebugAgent) sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (a *coopDebugAgent) logf(format string, args ...interface{}) {
	if a.out == nil {
		return
	}
	fmt.Fprintf(a.out, "[debug-agent] "+format+"\n", args...)
}
