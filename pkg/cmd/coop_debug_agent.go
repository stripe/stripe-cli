package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
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
	dc.cmd.Flags().MarkHidden("delay") //nolint:gosec

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
	a.log("debug agent attached to %s", a.sessionID)

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

		if step := firstStepWithState(session, coop.StepActive); step > 0 {
			if err := a.completeActiveStep(ctx, step); err != nil {
				return err
			}
			continue
		}

		if step := firstStepWithState(session, coop.StepReview); step > 0 {
			if shouldContinueChapterBeforeReview(session, step) {
				chapter, chapterIndex, _, err := session.ChapterByStepNumber(step)
				if err != nil {
					return err
				}
				if next := nextPendingStepInChapter(session, chapterIndex, step); next > 0 {
					a.log("section %q still has pending work; continuing with step %d", chapter.Title, next)
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

		if step := firstStepWithState(session, coop.StepPending); step > 0 {
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
	if node.State != coop.StepPending {
		return nil
	}

	if err := session.TransitionStep(step, coop.StepActive); err != nil {
		return err
	}
	node, _ = session.NodeByNumber(step)
	node.Activity = "Debug agent working: " + node.Title

	a.log("step %d active: %s", step, node.Title)
	if err := a.store.Write(session); err != nil {
		if isVersionConflict(err) {
			return nil
		}
		return err
	}
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
	if node.State != coop.StepActive {
		return nil
	}

	targetState := coop.StepReview
	if node.AutoConfirm || session.ReviewGranularityForStep(step) == coop.ReviewGranularityAuto {
		targetState = coop.StepDone
	}

	node.Verifications = append(node.Verifications, coop.Verification{
		Check:  "Debug agent deterministic check",
		Passed: true,
	})
	node.Implementation = &coop.Implementation{
		File:  "debug/" + safeDebugFileName(node.Key) + ".txt",
		Lines: "1-1",
		Note:  "Deterministic debug agent completed " + node.Title,
	}

	if err := session.TransitionStep(step, targetState); err != nil {
		return err
	}
	node.Activity = ""

	a.log("step %d %s: %s", step, targetState, node.Title)
	if err := a.store.Write(session); err != nil {
		if isVersionConflict(err) {
			return nil
		}
		return err
	}
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

	if chapterReviewApplies(session, step) {
		chapter, chapterIndex, _, err := session.ChapterByStepNumber(step)
		if err != nil {
			return err
		}
		a.log("waiting for section review: %s", chapter.Title)
		return a.awaitChapterReview(ctx, chapterIndex)
	}

	a.log("waiting for review: step %d %s", step, node.Title)
	for {
		a.store.WriteHeartbeat(a.sessionID)
		if err := a.sleep(ctx, a.pollInterval); err != nil {
			a.store.RemoveHeartbeat(a.sessionID)
			return err
		}

		session, err = a.store.Read(a.sessionID)
		if err != nil {
			a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		node, err = session.NodeByNumber(step)
		if err != nil {
			a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		if node.State != coop.StepReview {
			a.store.RemoveHeartbeat(a.sessionID)
			a.log("review released: step %d is %s", step, node.State)
			return nil
		}
	}
}

func (a *coopDebugAgent) awaitChapterReview(ctx context.Context, chapterIndex int) error {
	for {
		a.store.WriteHeartbeat(a.sessionID)
		if err := a.sleep(ctx, a.pollInterval); err != nil {
			a.store.RemoveHeartbeat(a.sessionID)
			return err
		}

		session, err := a.store.Read(a.sessionID)
		if err != nil {
			a.store.RemoveHeartbeat(a.sessionID)
			return err
		}
		if active := session.FirstActiveStepInChapter(chapterIndex); active > 0 {
			a.store.RemoveHeartbeat(a.sessionID)
			a.log("section requested changes; rerunning from step %d", active)
			return nil
		}
		if !session.ChapterHasReview(chapterIndex) {
			a.store.RemoveHeartbeat(a.sessionID)
			a.log("section review released")
			return nil
		}
	}
}

func (a *coopDebugAgent) completeSession(ctx context.Context, session *coop.Session) error {
	if session.Status != coop.SessionCompleted || session.NextSteps == nil || len(session.NextSteps.Suggestions) == 0 {
		suggestions := buildSuggestions(session, detectProjectEnvironment())
		var tuiSuggestions []coop.NextStepSuggestion
		for _, s := range suggestions {
			tuiSuggestions = append(tuiSuggestions, coop.NextStepSuggestion{
				ID:          s.ID,
				Title:       s.Title,
				Description: s.Description,
				Reason:      s.Reason,
			})
		}

		if session.NextSteps == nil {
			session.NextSteps = &coop.NextStepsState{}
		}
		session.NextSteps.Suggestions = tuiSuggestions
		session.NextSteps.Selected = ""
		session.Status = coop.SessionCompleted
		a.log("all steps complete; showing next steps")
		if err := a.store.Write(session); err != nil {
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
		a.log("next step selected: %s", selected)
		return nil
	}
}

func firstStepWithState(session *coop.Session, state coop.StepState) int {
	step := 0
	for i := range session.Chapters {
		for j := range session.Chapters[i].Nodes {
			step++
			if session.Chapters[i].Nodes[j].State == state {
				return step
			}
		}
	}
	return 0
}

func shouldContinueChapterBeforeReview(session *coop.Session, step int) bool {
	if !chapterReviewApplies(session, step) {
		return false
	}
	_, chapterIndex, _, err := session.ChapterByStepNumber(step)
	if err != nil {
		return false
	}
	return !session.ChapterReadyForReview(chapterIndex)
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

func (a *coopDebugAgent) log(format string, args ...interface{}) {
	if a.out == nil {
		return
	}
	fmt.Fprintf(a.out, "[debug-agent] "+format+"\n", args...)
}
