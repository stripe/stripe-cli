package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopNextStepsCmd struct {
	cmd       *cobra.Command
	session   string
	completed string
}

type suggestion struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Reason      string `json:"reason,omitempty"`
}

type nextStepsResponse struct {
	OK          bool         `json:"ok"`
	SessionID   string       `json:"session_id"`
	Completed   string       `json:"completed"`
	Suggestions []suggestion `json:"suggestions"`
	AgentPrompt string       `json:"agent_prompt"`
	Next        string       `json:"next"`
}

func newCoopNextStepsCmd() *coopNextStepsCmd {
	nc := &coopNextStepsCmd{}
	nc.cmd = &cobra.Command{
		Use:   "next-steps",
		Short: "Suggest what to do after completing an integration",
		Long: `Inspects the completed session and project environment to suggest
logical next steps: deploy, go live, add webhooks, or build more.

Automatically detects existing deploy targets, framework, and Stripe
Projects configuration to make relevant suggestions.`,
		RunE: nc.runNextStepsCmd,
	}

	nc.cmd.Flags().StringVar(&nc.session, "session", "", "Session ID (defaults to latest)")
	nc.cmd.Flags().StringVar(&nc.completed, "completed", "", "Mark a suggestion as completed (used by agent after finishing a task)")

	return nc
}

func (nc *coopNextStepsCmd) runNextStepsCmd(cmd *cobra.Command, args []string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var session *coop.Session
	if nc.session != "" {
		session, err = store.Read(nc.session)
	} else {
		session, err = store.LatestSession()
	}
	if err != nil {
		return outputCoopError("No session found.", "stripe coop run <blueprint>")
	}

	env := detectProjectEnvironment()
	suggestions := buildSuggestions(session, env)

	// Write suggestions to session so TUI renders the completion view
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
		return fmt.Errorf("writing next-step suggestions: %w", err)
	}

	if nc.completed != "" {
		session.NextSteps.Completed = append(session.NextSteps.Completed, nc.completed)
		if err := store.Write(session); err != nil {
			return fmt.Errorf("marking next-step completed: %w", err)
		}
	}

	// Block until the user selects something in the TUI
	for {
		time.Sleep(500 * time.Millisecond)
		session, err = store.Read(session.ID)
		if err != nil {
			continue
		}
		if session.NextSteps != nil && session.NextSteps.Selected != "" {
			break
		}
	}

	selected := session.NextSteps.Selected

	// Clear selection so we can await again if needed
	session.NextSteps.Selected = ""
	if err := store.Write(session); err != nil {
		return fmt.Errorf("clearing next-step selection: %w", err)
	}

	return outputJSON(buildNextStepsResponse(session, suggestions, selected))
}

func buildNextStepsResponse(session *coop.Session, suggestions []suggestion, selected string) nextStepsResponse {
	switch selected {
	case "summarize":
		return nextStepsResponse{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: buildSummarizePrompt(session),
			Next:        fmt.Sprintf("Write STRIPE.md, then run: stripe coop next-steps --session=%s --completed=summarize", session.ID),
		}
	case "deploy", "deploy-update":
		lang := session.Settings["language"]
		if lang == "" {
			lang = "node"
		}
		next := fmt.Sprintf("stripe coop run deploy-stripe-projects --language=%s --parent-session=%s --parent-step=%s", lang, session.ID, selected)
		return nextStepsResponse{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: "The developer wants to deploy. Start a new co-op session with the deploy blueprint.",
			Next:        next,
		}
	case "add-integration":
		return nextStepsResponse{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			Suggestions: suggestions,
			AgentPrompt: fmt.Sprintf("The developer wants to add another Stripe feature. Run 'stripe coop recommend' and ask what they need, then start a new session with --parent-session=%s --parent-step=add-integration.", session.ID),
			Next:        "stripe coop recommend",
		}
	case "done":
		return nextStepsResponse{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: "The developer is done. End the session.",
			Next:        fmt.Sprintf("stripe coop stop --session=%s", session.ID),
		}
	default:
		return nextStepsResponse{
			OK:          true,
			SessionID:   session.ID,
			Completed:   session.Blueprint,
			AgentPrompt: fmt.Sprintf("The developer selected: %s", selected),
			Next:        "stripe coop stop",
		}
	}
}
