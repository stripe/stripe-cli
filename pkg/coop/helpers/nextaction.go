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
var ErrSelectionTimeout = errors.New("timed out waiting for next-action selection")

const NextActionSelectionTimeout = 10 * time.Minute

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
	if err := ShowSuggestions(store, session, suggestions, input.Completed); err != nil {
		return Response{}, err
	}

	selected, err := WaitForSelection(store, session.ID)
	if err != nil {
		return Response{}, err
	}
	return BuildResponse(session, suggestions, selected), nil
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

	session.NextSteps.Suggestions = tuiSuggestions
	session.NextSteps.Selected = ""
	session.Status = coop.SessionCompleted
	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing next-action suggestions: %w", err)
	}
	return nil
}

func WaitForSelection(store Store, sessionID string) (string, error) {
	return waitForSelection(store, sessionID, NextActionSelectionTimeout, time.Now, time.Sleep)
}

func waitForSelection(store Store, sessionID string, timeout time.Duration, now func() time.Time, sleep func(time.Duration)) (string, error) {
	deadline := now().Add(timeout)
	for {
		if now().After(deadline) {
			return "", ErrSelectionTimeout
		}
		sleep(500 * time.Millisecond)
		session, err := store.Read(sessionID)
		if err != nil {
			continue
		}
		if session.NextSteps == nil || session.NextSteps.Selected == "" {
			continue
		}

		selected := session.NextSteps.Selected
		session.NextSteps.Selected = ""
		if err := store.Write(session); err != nil {
			return "", fmt.Errorf("clearing next-action selection: %w", err)
		}
		return selected, nil
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
