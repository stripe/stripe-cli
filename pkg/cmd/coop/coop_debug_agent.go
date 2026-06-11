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
				if next := helpers.NextPendingStepInChapter(session, chapterIndex, step); next > 0 {
					a.logf("section %q still has pending work; continuing with step %d", chapter.Title, next)
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
	if node.State != coop.StepActive {
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

	if helpers.ChapterReviewApplies(session, step) {
		chapter, chapterIndex, _, err := session.ChapterByStepNumber(step)
		if err != nil {
			return err
		}
		a.logf("waiting for section review: %s", chapter.Title)
		return a.awaitChapterReview(ctx, chapterIndex)
	}

	a.logf("waiting for review: step %d %s", step, node.Title)
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
			a.logf("review released: step %d is %s", step, node.State)
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
			a.logf("section requested changes; rerunning from step %d", active)
			return nil
		}
		if !session.ChapterHasReview(chapterIndex) {
			a.store.RemoveHeartbeat(a.sessionID)
			a.logf("section review released")
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
	if !helpers.ChapterReviewApplies(session, step) {
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

func (a *coopDebugAgent) logf(format string, args ...interface{}) {
	if a.out == nil {
		return
	}
	fmt.Fprintf(a.out, "[debug-agent] "+format+"\n", args...)
}
