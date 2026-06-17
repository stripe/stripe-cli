// Package workflow applies co-op agent lifecycle transitions to sessions.
package workflow

import (
	"fmt"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/helpers"
)

const AwaitTimeout = 10 * time.Minute

type Store interface {
	Read(id string) (*coop.Session, error)
	Update(id string, fn func(*coop.Session) error) (*coop.Session, error)
	WriteHeartbeat(id string)
	RemoveHeartbeat(id string)
}

type Service struct {
	store        Store
	fetchSnippet func(path, method string, params interface{}, language string) (string, error)
	now          func() time.Time
	sleep        func(time.Duration)
	awaitTimeout time.Duration
}

type Option func(*Service)

func WithSnippetFetcher(fetch func(path, method string, params interface{}, language string) (string, error)) Option {
	return func(s *Service) {
		s.fetchSnippet = fetch
	}
}

func WithClock(now func() time.Time, sleep func(time.Duration)) Option {
	return func(s *Service) {
		if now != nil {
			s.now = now
		}
		if sleep != nil {
			s.sleep = sleep
		}
	}
}

func WithAwaitTimeout(timeout time.Duration) Option {
	return func(s *Service) {
		s.awaitTimeout = timeout
	}
}

func NewService(store Store, opts ...Option) *Service {
	s := &Service{
		store:        store,
		fetchSnippet: coop.FetchSDKSnippet,
		now:          time.Now,
		sleep:        time.Sleep,
		awaitTimeout: AwaitTimeout,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ReportWorkInput struct {
	File    string
	Lines   string
	Snippet string
	Note    string
}

func (s *Service) StartWork(sessionID string, step int, note string) (coop.CommandResponse, error) {
	session, err := s.store.Update(sessionID, func(session *coop.Session) error {
		if err := session.TransitionStep(step, coop.StepActive); err != nil {
			return err
		}
		node, _ := session.NodeByNumber(step)
		node.Activity = note
		return nil
	})
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}

	node, _ := session.NodeByNumber(step)
	resp := coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     string(coop.StepActive),
		Message:   fmt.Sprintf("Started: %s", node.Title),
		Next:      fmt.Sprintf("stripe coop agent report-work --session=%s --step=%d --file=<path> --note=\"<what you did>\"", session.ID, step),
	}
	if node.Type == coop.NodeAPIRequest && node.Request != nil {
		resp.APIRequest = node.Request
		if snippet, err := s.fetchSnippet(node.Request.Path, node.Request.Method, node.Request.Params, language(session)); err == nil {
			resp.SDKExample = snippet
		}
	}
	return resp, nil
}

func (s *Service) ReportWork(sessionID string, step int, input ReportWorkInput, autoConfirm bool) (coop.CommandResponse, error) {
	var targetState coop.StepState
	session, err := s.store.Update(sessionID, func(session *coop.Session) error {
		node, err := session.NodeByNumber(step)
		if err != nil {
			return err
		}
		targetState = coop.StepReview
		if autoConfirm || node.AutoConfirm || session.ReviewGranularityForStep(step) == coop.ReviewGranularityAuto {
			targetState = coop.StepDone
		}
		if err := session.TransitionStep(step, targetState); err != nil {
			return err
		}
		node, _ = session.NodeByNumber(step)
		if input.File != "" || input.Snippet != "" || input.Note != "" {
			node.Implementation = &coop.Implementation{
				File:    input.File,
				Lines:   input.Lines,
				Snippet: input.Snippet,
				Note:    input.Note,
			}
		}
		node.Activity = ""
		if session.IsComplete() {
			session.Status = coop.SessionCompleted
		}
		return nil
	})
	if err != nil {
		return errorResponse(err, fmt.Sprintf("stripe coop agent start-work --session=%s --step=%d", sessionID, step)), nil
	}
	node, _ := session.NodeByNumber(step)
	return s.reportWorkResponse(session, node, step, targetState), nil
}

func (s *Service) ReportCheck(sessionID string, step int, check string, passed bool) (coop.CommandResponse, error) {
	if strings.TrimSpace(check) == "" {
		return errorResponse(fmt.Errorf("--check flag is required"), fmt.Sprintf("stripe coop agent report-check --session=%s --step=%d --check=\"<label>\" --passed", sessionID, step)), nil
	}
	session, err := s.store.Update(sessionID, func(session *coop.Session) error {
		node, err := session.NodeByNumber(step)
		if err != nil {
			return err
		}
		node.Verifications = append(node.Verifications, coop.Verification{Check: check, Passed: passed})
		return nil
	})
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}
	node, _ := session.NodeByNumber(step)
	status := "failed"
	if passed {
		status = "passed"
	}
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     string(node.State),
		Message:   fmt.Sprintf("Verification %s: %s", status, check),
		Next:      fmt.Sprintf("stripe coop agent report-work --session=%s --step=%d --file=<path> --note=\"<what you did>\"", session.ID, step),
	}, nil
}

func (s *Service) Skip(sessionID string, step int, note string) (coop.CommandResponse, error) {
	session, err := s.store.Update(sessionID, func(session *coop.Session) error {
		if err := session.TransitionStep(step, coop.StepSkipped); err != nil {
			return err
		}
		node, _ := session.NodeByNumber(step)
		node.Activity = note
		if session.IsComplete() {
			session.Status = coop.SessionCompleted
		}
		return nil
	})
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}
	node, _ := session.NodeByNumber(step)
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     string(coop.StepSkipped),
		Message:   fmt.Sprintf("Skipped: %s", node.Title),
		Next:      nextAfterStep(session, step),
	}, nil
}

func (s *Service) ConfirmReview(sessionID string, steps []int) (*coop.Session, error) {
	return s.store.Update(sessionID, func(session *coop.Session) error {
		for _, step := range steps {
			node, err := session.NodeByNumber(step)
			if err != nil {
				return err
			}
			if node.State == coop.StepDone {
				continue
			}
			if err := session.TransitionStep(step, coop.StepDone); err != nil {
				return err
			}
		}
		if session.IsComplete() {
			session.Status = coop.SessionCompleted
		}
		return nil
	})
}

func (s *Service) RequestChanges(sessionID string, steps []int, note string) (*coop.Session, error) {
	if strings.TrimSpace(note) == "" {
		return nil, fmt.Errorf("request changes note is required")
	}
	return s.store.Update(sessionID, func(session *coop.Session) error {
		for _, step := range steps {
			node, err := session.NodeByNumber(step)
			if err != nil {
				return err
			}
			if node.State != coop.StepActive {
				if err := session.TransitionStep(step, coop.StepActive); err != nil {
					return err
				}
				node, _ = session.NodeByNumber(step)
			}
			node.RejectionNote = note
			node.Implementation = nil
			node.Verifications = nil
		}
		return nil
	})
}

func (s *Service) AwaitReview(sessionID string, step int) (coop.CommandResponse, error) {
	session, err := s.store.Read(sessionID)
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}
	node, err := session.NodeByNumber(step)
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}

	if node.AutoConfirm && node.State == coop.StepReview {
		return s.autoConfirm(sessionID, step)
	}
	if helpers.ChapterReviewApplies(session, step) && node.State == coop.StepReview {
		chapter, chapterIndex, _, err := session.ChapterByStepNumber(step)
		if err != nil {
			return errorResponse(err, "stripe coop status"), nil
		}
		if !session.ChapterReadyForReview(chapterIndex) {
			return coop.CommandResponse{
				OK:        true,
				SessionID: session.ID,
				Step:      step,
				State:     string(coop.StepReview),
				Message:   fmt.Sprintf("Step %d is ready. Continue the section before asking for human review.", step),
				Next:      nextInChapterOrStatus(session, chapterIndex, step),
			}, nil
		}
		return s.awaitChapterReview(session.ID, chapter.Title, chapterIndex, step)
	}
	if node.State != coop.StepReview {
		return alreadyMovedResponse(session, step, node.State), nil
	}
	return s.awaitStepReview(session.ID, step)
}

func (s *Service) SelectNextAction(sessionID, selected, completed string, suggestions []coop.NextStepSuggestion) (*coop.Session, error) {
	return s.store.Update(sessionID, func(session *coop.Session) error {
		if session.NextSteps == nil {
			session.NextSteps = &coop.NextStepsState{}
		}
		if len(suggestions) > 0 {
			session.NextSteps.Suggestions = suggestions
		}
		if completed != "" && !contains(session.NextSteps.Completed, completed) {
			session.NextSteps.Completed = append(session.NextSteps.Completed, completed)
		}
		session.NextSteps.Selected = selected
		session.Status = coop.SessionCompleted
		return nil
	})
}

func (s *Service) autoConfirm(sessionID string, step int) (coop.CommandResponse, error) {
	session, err := s.ConfirmReview(sessionID, []int{step})
	if err != nil {
		return errorResponse(err, "stripe coop status"), nil
	}
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     "confirmed",
		Message:   fmt.Sprintf("Step %d auto-confirmed. Proceed to next step.", step),
		Next:      nextAfterStep(session, step),
	}, nil
}

func (s *Service) awaitStepReview(sessionID string, step int) (coop.CommandResponse, error) {
	s.store.WriteHeartbeat(sessionID)
	defer s.store.RemoveHeartbeat(sessionID)

	deadline := s.now().Add(s.awaitTimeout)
	for {
		if s.now().After(deadline) {
			return timeoutResponse(sessionID, step), nil
		}
		s.sleep(500 * time.Millisecond)
		s.store.WriteHeartbeat(sessionID)

		session, err := s.store.Read(sessionID)
		if err != nil {
			return coop.CommandResponse{}, err
		}
		node, err := session.NodeByNumber(step)
		if err != nil {
			return coop.CommandResponse{}, err
		}
		if node.State == coop.StepReview {
			continue
		}
		if node.State == coop.StepDone {
			return confirmedResponse(session, step), nil
		}
		if node.State == coop.StepActive {
			return rejectedResponse(session, step, node), nil
		}
		return alreadyMovedResponse(session, step, node.State), nil
	}
}

func (s *Service) awaitChapterReview(sessionID, chapterTitle string, chapterIndex, step int) (coop.CommandResponse, error) {
	s.store.WriteHeartbeat(sessionID)
	defer s.store.RemoveHeartbeat(sessionID)

	deadline := s.now().Add(s.awaitTimeout)
	for {
		if s.now().After(deadline) {
			return timeoutResponse(sessionID, step), nil
		}
		s.sleep(500 * time.Millisecond)
		s.store.WriteHeartbeat(sessionID)

		session, err := s.store.Read(sessionID)
		if err != nil {
			return coop.CommandResponse{}, err
		}
		if activeStep := session.FirstActiveStepInChapter(chapterIndex); activeStep > 0 {
			activeNode, _ := session.NodeByNumber(activeStep)
			msg := fmt.Sprintf("Section %q requested changes.", chapterTitle)
			if activeNode != nil && activeNode.RejectionNote != "" {
				msg += fmt.Sprintf("\nFeedback: %s", activeNode.RejectionNote)
			}
			msg += "\nRedo the section from the first affected step."
			return coop.CommandResponse{
				OK:        true,
				SessionID: session.ID,
				Step:      activeStep,
				State:     "rejected",
				Message:   msg,
				Next:      fmt.Sprintf("stripe coop agent start-work --session=%s --step=%d --note=%s", session.ID, activeStep, quoteArg("Redoing: "+activeNode.Title)),
			}, nil
		}
		if session.ChapterHasReview(chapterIndex) {
			continue
		}
		return confirmedResponse(session, step), nil
	}
}

func (s *Service) reportWorkResponse(session *coop.Session, node *coop.SessionNode, step int, targetState coop.StepState) coop.CommandResponse {
	if targetState == coop.StepReview {
		if helpers.ChapterReviewApplies(session, step) {
			chapter, chapterIndex, _, err := session.ChapterByStepNumber(step)
			if err == nil && !session.ChapterReadyForReview(chapterIndex) {
				return coop.CommandResponse{
					OK:        true,
					SessionID: session.ID,
					Step:      step,
					State:     string(coop.StepReview),
					Message:   fmt.Sprintf("Ready: %s. Continue the section before asking for human review.", node.Title),
					Next:      nextInChapterOrStatus(session, chapterIndex, step),
				}
			}
			if err == nil {
				return coop.CommandResponse{
					OK:        true,
					SessionID: session.ID,
					Step:      step,
					State:     string(coop.StepReview),
					Message:   fmt.Sprintf("Section ready for review: %s. Run relevant checks, keep useful servers running, share local URLs or test data, then await review.", chapter.Title),
					Next:      fmt.Sprintf("stripe coop agent await-review --session=%s --step=%d", session.ID, step),
				}
			}
		}
		return coop.CommandResponse{
			OK:        true,
			SessionID: session.ID,
			Step:      step,
			State:     string(coop.StepReview),
			Message:   fmt.Sprintf("Ready for review: %s", node.Title),
			Next:      fmt.Sprintf("stripe coop agent await-review --session=%s --step=%d", session.ID, step),
		}
	}

	msg := fmt.Sprintf("Completed: %s", node.Title)
	next := nextAfterStep(session, step)
	if session.IsComplete() {
		msg += " All steps complete. Run next-action so the developer can choose what happens next."
	}
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     string(targetState),
		Message:   msg,
		Next:      next,
	}
}

func nextAfterStep(session *coop.Session, step int) string {
	if nextStep := session.NextPendingStep(step); nextStep > 0 {
		nextNode, _ := session.NodeByNumber(nextStep)
		return fmt.Sprintf("stripe coop agent start-work --session=%s --step=%d --note=%s", session.ID, nextStep, quoteArg("Beginning: "+nextNode.Title))
	}
	if session.IsComplete() {
		return fmt.Sprintf("stripe coop agent next-action --session=%s", session.ID)
	}
	return fmt.Sprintf("stripe coop status --session=%s", session.ID)
}

func nextInChapterOrStatus(session *coop.Session, chapterIndex, afterStep int) string {
	if nextStep := helpers.NextPendingStepInChapter(session, chapterIndex, afterStep); nextStep > 0 {
		nextNode, _ := session.NodeByNumber(nextStep)
		return fmt.Sprintf("stripe coop agent start-work --session=%s --step=%d --note=%s", session.ID, nextStep, quoteArg("Beginning: "+nextNode.Title))
	}
	return fmt.Sprintf("stripe coop status --session=%s", session.ID)
}

func alreadyMovedResponse(session *coop.Session, step int, state coop.StepState) coop.CommandResponse {
	msg := fmt.Sprintf("Step %d is already %s.", step, state)
	if session.IsComplete() {
		msg = fmt.Sprintf("Step %d confirmed. All steps done. Run next-action now.", step)
	}
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     string(state),
		Message:   msg,
		Next:      nextAfterStep(session, step),
	}
}

func confirmedResponse(session *coop.Session, step int) coop.CommandResponse {
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     "confirmed",
		Message:   fmt.Sprintf("Step %d confirmed by developer. Proceed to next step.", step),
		Next:      nextAfterStep(session, step),
	}
}

func rejectedResponse(session *coop.Session, step int, node *coop.SessionNode) coop.CommandResponse {
	msg := fmt.Sprintf("Step %d rejected by developer.", step)
	if node.RejectionNote != "" {
		msg += fmt.Sprintf("\nFeedback: %s", node.RejectionNote)
	}
	msg += "\nRedo the step."
	return coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		Step:      step,
		State:     "rejected",
		Message:   msg,
		Next:      fmt.Sprintf("stripe coop agent report-work --session=%s --step=%d --file=<path> --note=\"<what you fixed>\"", session.ID, step),
	}
}

func timeoutResponse(sessionID string, step int) coop.CommandResponse {
	return coop.CommandResponse{
		OK:        true,
		SessionID: sessionID,
		Step:      step,
		State:     "timeout",
		Message:   "Timed out waiting for developer confirmation. Re-run await-review to wait again.",
		Next:      fmt.Sprintf("stripe coop agent await-review --session=%s --step=%d", sessionID, step),
	}
}

func errorResponse(err error, hint string) coop.CommandResponse {
	return coop.CommandResponse{OK: false, Error: err.Error(), Hint: hint}
}

func language(session *coop.Session) string {
	if session != nil && session.Settings != nil && session.Settings["language"] != "" {
		return session.Settings["language"]
	}
	return "node"
}

func quoteArg(value string) string {
	return fmt.Sprintf("%q", value)
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
