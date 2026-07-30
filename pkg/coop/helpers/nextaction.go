// Package helpers contains shared support logic for co-op commands and workflows.
package helpers

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
)

var ErrNoSession = errors.New("no session found")

// NextActionInterval mirrors workflow.AwaitTimeout: one next-action call waits
// at most this long, then returns a waiting response asking the agent to run it
// again. It must stay under an agent harness's default command timeout for the
// same reason await-review must — a killed process hands the agent no next
// command. (Declared here rather than imported: workflow depends on helpers.)
const NextActionInterval = 45 * time.Second

// NextActionHarnessTimeout is the shell timeout advertised to agents.
const NextActionHarnessTimeout = 90 * time.Second

type Input struct {
	SessionID string
	Completed string
}

type Store interface {
	Read(id string) (*coop.Session, error)
	LatestSession() (*coop.Session, error)
	Write(session *coop.Session) error
}

type Suggestion struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason,omitempty"`
}

type Response struct {
	OK          bool         `json:"ok"`
	SessionID   string       `json:"session_id"`
	Completed   string       `json:"completed"`
	Suggestions []Suggestion `json:"suggestions"`
	AgentPrompt string       `json:"agent_prompt"`

	State          string `json:"state,omitempty"`
	AdvanceAllowed *bool  `json:"advance_allowed,omitempty"`
	WaitedSeconds  int    `json:"waited_seconds,omitempty"`
	Message        string `json:"message,omitempty"`

	coop.Continuation
}

type Environment struct {
	HasStripeProjects bool
	HasVercel         bool
	HasFly            bool
	HasNetlify        bool
	HasExistingDeploy bool
}

func Run(store Store, input Input) (Response, error) {
	var session *coop.Session
	var err error
	if input.SessionID != "" {
		session, err = store.Read(input.SessionID)
	} else {
		session, err = store.LatestSession()
	}
	if err != nil {
		return Response{}, ErrNoSession
	}

	suggestions := filterCompletedSuggestions(
		BuildSuggestions(session, DetectProjectEnvironment()),
		completedActionIDs(session, input.Completed),
	)

	// Consume a selection made between invocations BEFORE republishing.
	// ShowSuggestions clears NextSteps.Selected, so publishing first would erase
	// a choice the developer made while the previous invocation was exiting,
	// leaving the agent waiting on a selection that no longer exists.
	if selected := pendingSelection(session); selected != "" {
		if err := consumeSelection(store, session); err != nil {
			return Response{}, err
		}
		return BuildResponse(session, suggestions, selected), nil
	}

	if err := ShowSuggestions(store, session, suggestions, input.Completed); err != nil {
		return Response{}, err
	}

	selected, waited, err := waitForSelection(store, session.ID, NextActionInterval, time.Now, time.Sleep)
	if err != nil {
		return Response{}, err
	}
	if selected == "" {
		return waitingResponse(session, suggestions, input.Completed, waited), nil
	}
	return BuildResponse(session, suggestions, selected), nil
}

// waitingResponse says the developer has not chosen yet. Like await-review it
// exits successfully and repeats the invoked command, so an agent never reads a
// still-open choice as a broken session.
func waitingResponse(session *coop.Session, suggestions []Suggestion, completed string, waited time.Duration) Response {
	return Response{
		OK:             true,
		SessionID:      session.ID,
		Completed:      session.Blueprint,
		Suggestions:    suggestions,
		State:          "waiting",
		AdvanceAllowed: advanceAllowed(false),
		WaitedSeconds:  int(waited.Round(time.Second).Seconds()),
		Message: "The developer has not picked what happens next yet. This is expected and nothing is wrong.\n" +
			"Do not end the session and do not ask a question here. Run the command in \"next\" again now to keep waiting.",
		Continuation: coop.Continue(coop.NextActionCommand(session.ID, completed)).
			WithWaitTimeout(int(NextActionInterval.Seconds())),
	}
}

func advanceAllowed(v bool) *bool {
	return &v
}

func ShowSuggestions(store Store, session *coop.Session, suggestions []Suggestion, completed string) error {
	if session.NextSteps == nil {
		session.NextSteps = &coop.NextStepsState{}
	}
	if completed != "" && !containsString(session.NextSteps.Completed, completed) {
		session.NextSteps.Completed = append(session.NextSteps.Completed, completed)
	}
	suggestions = filterCompletedSuggestions(suggestions, completedActionIDs(session, ""))

	var tuiSuggestions []coop.NextStepSuggestion
	for _, s := range suggestions {
		tuiSuggestions = append(tuiSuggestions, coop.NextStepSuggestion{
			ID:          s.ID,
			Title:       s.Title,
			Description: s.Description,
			Reason:      s.Reason,
		})
	}

	// Skip a no-op write. Republishing an identical suggestion set on every
	// call would bump the session version each time, which churns the TUI and
	// (because the stop hook keys its block budget on progress) would make the
	// hook's escape hatch unreachable.
	if suggestionsUnchanged(session, tuiSuggestions) {
		return nil
	}

	session.NextSteps.Suggestions = tuiSuggestions
	session.NextSteps.Selected = ""
	session.Status = coop.SessionCompleted
	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing next-action suggestions: %w", err)
	}
	return nil
}

func suggestionsUnchanged(session *coop.Session, next []coop.NextStepSuggestion) bool {
	if session.Status != coop.SessionCompleted || session.NextSteps.Selected != "" {
		return false
	}
	current := session.NextSteps.Suggestions
	if len(current) != len(next) {
		return false
	}
	for i := range current {
		if current[i] != next[i] {
			return false
		}
	}
	return true
}

// pendingSelection reports a choice the TUI already recorded.
func pendingSelection(session *coop.Session) string {
	if session == nil || session.NextSteps == nil || len(session.NextSteps.Suggestions) == 0 {
		return ""
	}
	return session.NextSteps.Selected
}

func consumeSelection(store Store, session *coop.Session) error {
	session.NextSteps.Selected = ""
	if err := store.Write(session); err != nil {
		return fmt.Errorf("clearing next-action selection: %w", err)
	}
	return nil
}

// WaitForSelection blocks for one interval. An empty selection with a nil error
// means the developer simply has not chosen yet.
func WaitForSelection(store Store, sessionID string) (string, error) {
	selected, _, err := waitForSelection(store, sessionID, NextActionInterval, time.Now, time.Sleep)
	return selected, err
}

func waitForSelection(store Store, sessionID string, timeout time.Duration, now func() time.Time, sleep func(time.Duration)) (string, time.Duration, error) {
	start := now()
	deadline := start.Add(timeout)
	for {
		// Poll before checking the deadline so a very short interval still
		// performs one read: a choice can land while the process starts.
		sleep(500 * time.Millisecond)
		session, err := store.Read(sessionID)
		if err == nil && session.NextSteps != nil && session.NextSteps.Selected != "" {
			selected := session.NextSteps.Selected
			if err := consumeSelection(store, session); err != nil {
				return "", now().Sub(start), err
			}
			return selected, now().Sub(start), nil
		}
		if now().After(deadline) {
			return "", now().Sub(start), nil
		}
	}
}

func BuildSuggestions(session *coop.Session, env Environment) []Suggestion {
	var suggestions []Suggestion

	switch {
	case env.HasStripeProjects:
		suggestions = append(suggestions, Suggestion{
			ID:          "deploy",
			Title:       "Deploy with Stripe Projects",
			Description: "Your project is already configured — run stripe projects deploy",
			Available:   true,
			Reason:      "stripe.json found",
		})
	case !env.HasExistingDeploy:
		suggestions = append(suggestions, Suggestion{
			ID:          "deploy",
			Title:       "Deploy with Stripe Projects",
			Description: "Set up hosting, CI/CD, and environment management",
			Available:   true,
			Reason:      "No deploy configuration detected",
		})
	default:
		target := env.deployTarget()
		suggestions = append(suggestions, Suggestion{
			ID:          "deploy-update",
			Title:       "Deploy your changes",
			Description: fmt.Sprintf("Push your new integration code to %s", target),
			Available:   true,
			Reason:      fmt.Sprintf("Detected: %s", target),
		})
	}

	suggestions = append(suggestions, Suggestion{
		ID:          "summarize",
		Title:       "Write a STRIPE.md summary",
		Description: "Generate a STRIPE.md with what was built, API resources created, environment setup, and how to run",
		Available:   true,
	})

	suggestions = append(suggestions, Suggestion{
		ID:          "add-integration",
		Title:       "Add another Stripe feature",
		Description: "Subscriptions, Connect, billing portal, and more",
		Available:   true,
	})

	suggestions = append(suggestions, Suggestion{
		ID:          "done",
		Title:       "Finish",
		Description: "Close this session",
		Available:   true,
	})

	return filterCompletedSuggestions(suggestions, completedActionIDs(session, ""))
}

func completedActionIDs(session *coop.Session, current string) map[string]bool {
	completed := map[string]bool{}
	if session != nil && session.NextSteps != nil {
		for _, id := range session.NextSteps.Completed {
			completed[id] = true
		}
	}
	if current != "" {
		completed[current] = true
	}
	return completed
}

func filterCompletedSuggestions(suggestions []Suggestion, completed map[string]bool) []Suggestion {
	if len(completed) == 0 {
		return suggestions
	}
	// Allocate a fresh slice rather than reusing the input's backing array
	// (suggestions[:0]), which would mutate the caller's slice in place.
	filtered := make([]Suggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if completed[suggestion.ID] {
			continue
		}
		filtered = append(filtered, suggestion)
	}
	return filtered
}

func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func BuildResponse(session *coop.Session, suggestions []Suggestion, selected string) Response {
	switch selected {
	case "summarize":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: BuildSummarizePrompt(session),
			Continuation: coop.Continue(
				coop.NextActionCommand(session.ID, "summarize"),
			),
		}
	case "deploy":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: BuildDeployPrompt(session),
			Continuation: coop.Continue(
				coop.StartFollowupCommand(session.ID, "deploy", ""),
			),
		}
	case "deploy-update":
		target := deployTargetFromSuggestion(suggestions, selected)
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: BuildDeployUpdatePrompt(session, target),
			Continuation: coop.Continue(
				coop.StartFollowupCommand(session.ID, "deploy-update", target),
			),
		}
	case "add-integration":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: fmt.Sprintf("The developer wants to add another Stripe feature. Run 'stripe coop recommend' and ask what they need, then start a new session with --parent-session=%s --parent-step=add-integration.", session.ID),
			Continuation: coop.Continue(
				"stripe coop recommend",
			),
		}
	case "done":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: "The developer is done. End the session.",
			Continuation: coop.Continue(
				coop.StopCommand(session.ID),
			),
		}
	default:
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: fmt.Sprintf("The developer selected: %s", selected),
			Continuation: coop.Continue(
				coop.StopCommand(""),
			),
		}
	}
}

func deployTargetFromSuggestion(suggestions []Suggestion, selected string) string {
	for _, suggestion := range suggestions {
		if suggestion.ID != selected {
			continue
		}
		target := strings.TrimPrefix(suggestion.Reason, "Detected: ")
		if target != suggestion.Reason && target != "" {
			return target
		}
		target = strings.TrimPrefix(suggestion.Description, "Push your new integration code to ")
		if target != suggestion.Description && target != "" {
			return target
		}
	}
	return "the detected deployment target"
}

func BuildDeployPrompt(session *coop.Session) string {
	return fmt.Sprintf(`The developer wants a guided deploy flow.

Start an internal deploy follow-up session by running the next command exactly as written. Do not use "stripe coop run"; deploy follow-ups are not co-op blueprints.

The guided session will show the step-by-step deploy work in the developer's TUI and will use Stripe Projects as the deployment source of truth.

Parent session: %s`, session.ID)
}

func BuildDeployUpdatePrompt(session *coop.Session, target string) string {
	return fmt.Sprintf(`The developer wants a guided deploy-update flow for %s.

Start an internal deploy-update follow-up session by running the next command exactly as written. Do not use "stripe coop run"; deploy follow-ups are not co-op blueprints.

The guided session will show the step-by-step deploy work in the developer's TUI and will use the existing %s deployment configuration.

Parent session: %s`, target, target, session.ID)
}

func BuildSummarizePrompt(session *coop.Session) string {
	return fmt.Sprintf(`The developer wants a STRIPE.md summary. Create a STRIPE.md file in the project root with:

## What was built
- Integration: %s
- Blueprint steps completed

## Stripe resources created
- List any product IDs, price IDs, customer IDs created during the session

## Environment variables
- STRIPE_SECRET_KEY — your Stripe test secret key
- STRIPE_WEBHOOK_SECRET — webhook signing secret (from stripe listen)

## How to run
- Commands to install deps and start the server

## Webhook events handled
- List the events this integration listens for

## Useful links
- Stripe Dashboard: https://dashboard.stripe.com/test
- API docs: https://docs.stripe.com/api

After writing the file, run "stripe coop agent next-action --session=%s --completed=summarize" again to offer more options.`, session.Blueprint, session.ID)
}

func DetectProjectEnvironment() Environment {
	env := Environment{}
	env.HasStripeProjects = fileExists("stripe.json") || dirExists(".stripe")
	env.HasVercel = fileExists("vercel.json") || fileExists(".vercel/project.json")
	env.HasFly = fileExists("fly.toml")
	env.HasNetlify = fileExists("netlify.toml")
	hasDocker := fileExists("Dockerfile") || fileExists("docker-compose.yml") || fileExists("docker-compose.yaml")
	hasRailway := fileExists("railway.json") || fileExists("railway.toml")
	env.HasExistingDeploy = env.HasVercel || env.HasFly || hasDocker || hasRailway || env.HasNetlify
	return env
}

func (env Environment) deployTarget() string {
	switch {
	case env.HasVercel:
		return "Vercel"
	case env.HasFly:
		return "Fly.io"
	case env.HasNetlify:
		return "Netlify"
	default:
		return "existing infrastructure"
	}
}

func fileExists(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func dirExists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.IsDir()
}
