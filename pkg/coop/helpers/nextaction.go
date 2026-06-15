// Package helpers contains shared support logic for co-op commands and workflows.
package helpers

import (
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
)

var ErrNoSession = errors.New("no session found")

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
	Next        string       `json:"next"`
}

type Environment struct {
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

	suggestions := BuildSuggestions(session, DetectProjectEnvironment())
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
	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing next-action suggestions: %w", err)
	}

	if completed != "" {
		session.NextSteps.Completed = append(session.NextSteps.Completed, completed)
		if err := store.Write(session); err != nil {
			return fmt.Errorf("marking next-action completed: %w", err)
		}
	}
	return nil
}

func WaitForSelection(store Store, sessionID string) (string, error) {
	for {
		time.Sleep(500 * time.Millisecond)
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

	return suggestions
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
			Next:        fmt.Sprintf("Write STRIPE.md, then run: stripe coop agent next-action --session=%s --completed=summarize", session.ID),
		}
	case "add-integration":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: fmt.Sprintf("The developer wants to add another Stripe feature. Run 'stripe coop recommend' and ask what they need, then start a new session with --parent-session=%s --parent-step=add-integration.", session.ID),
			Next:        "stripe coop recommend",
		}
	case "done":
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: "The developer is done. End the session.",
			Next:        fmt.Sprintf("stripe coop stop --session=%s", session.ID),
		}
	default:
		return Response{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: fmt.Sprintf("The developer selected: %s", selected),
			Next:        "stripe coop stop",
		}
	}
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
	return Environment{}
}
