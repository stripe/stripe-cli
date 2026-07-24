package coopcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopAgentRunCmd struct {
	cmd           *cobra.Command
	language      string
	settings      []string
	params        []string
	parentSession string
	parentStep    string
}

func newCoopAgentRunCmd() *coopAgentRunCmd {
	rc := &coopAgentRunCmd{}
	rc.cmd = &cobra.Command{
		Use:   "run <blueprint-id>",
		Short: "Create a co-op session from a blueprint (agent-facing)",
		Long: `Creates a new co-op session using the specified blueprint. The session file
is written to disk and the agent can immediately begin working through nodes.

This is the agent-facing command. Developers should use "stripe coop start" instead.`,
		Example: `  stripe coop run one-time-payment
  stripe coop run one-time-payment --language=node
  stripe coop run setup-future-payments --setting=framework=express --param=customer_type=existing`,
		Args: cobra.ExactArgs(1),
		RunE: rc.runCmd,
	}

	rc.cmd.Flags().StringVar(&rc.language, "language", "", "Programming language for the integration")
	rc.cmd.Flags().StringArrayVar(&rc.settings, "setting", nil, "Blueprint settings as key=value pairs")
	rc.cmd.Flags().StringArrayVar(&rc.params, "param", nil, "Blueprint params as key=value pairs")
	rc.cmd.Flags().StringVar(&rc.parentSession, "parent-session", "", "Parent co-op session ID for follow-up work")
	rc.cmd.Flags().StringVar(&rc.parentStep, "parent-step", "", "Parent next-step ID this session fulfills")

	return rc
}

func (rc *coopAgentRunCmd) runCmd(cmd *cobra.Command, args []string) error {
	blueprintID := args[0]

	bp, err := coop.LoadBlueprint(blueprintID)
	if err != nil {
		// Surface the specific error (e.g. an ambiguous prefix and its candidate
		// list) rather than a generic "not found".
		return outputCoopError(err.Error(), "stripe coop recommend")
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	sessionID := "coop_" + uuid.New().String()[:8]

	session, err := newCoopSession(bp, sessionID, rc.language, rc.settings, rc.params, rc.parentSession, rc.parentStep)
	if err != nil {
		return outputCoopError(err.Error(), "Use --setting key=value and --param key=value.")
	}

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	resp := newCoopAgentRunResponse(bp, session)

	return outputJSON(resp)
}

func newCoopAgentRunResponse(bp *coop.Blueprint, session *coop.Session) coop.CommandResponse {
	return newCoopAgentSessionResponse(bp.Title, session, agentInstructions(bp))
}

func newCoopAgentGuidedActionResponse(action *coop.GuidedAction, session *coop.Session) coop.CommandResponse {
	return newCoopAgentSessionResponse(action.Title, session, guidedActionAgentInstructions(action))
}

func newCoopAgentSessionResponse(title string, session *coop.Session, instructions string) coop.CommandResponse {
	return coop.CommandResponse{
		OK:          true,
		SessionID:   session.ID,
		Node:        1,
		State:       "created",
		Message:     fmt.Sprintf("Session started: %s (%d nodes)", title, session.TotalNodes()),
		Next:        fmt.Sprintf("stripe coop agent start-work --session=%s --step=1 --note=%s", session.ID, quoteArg("Beginning: "+session.Steps[0].Nodes[0].Title)),
		AgentPrompt: instructions,
	}
}

func newCoopSession(bp *coop.Blueprint, sessionID, language string, rawSettings, rawParams []string, parentSession, parentStep string) (*coop.Session, error) {
	settings := make(map[string]string)
	if language != "" {
		settings["language"] = language
	}
	if err := mergeKeyValues(settings, "--setting", rawSettings); err != nil {
		return nil, err
	}

	params := make(map[string]string)
	if err := mergeKeyValues(params, "--param", rawParams); err != nil {
		return nil, err
	}

	session := coop.NewSessionFromBlueprint(bp, sessionID, settings, params)
	session.CreatedAt = time.Now().UTC()
	session.ParentSessionID = parentSession
	session.ParentStepID = parentStep
	session.UsedSandbox = coopSandboxClaimURL() != ""
	return session, nil
}

func mergeKeyValues(dst map[string]string, flag string, values []string) error {
	for _, value := range values {
		key, val, ok := strings.Cut(value, "=")
		if !ok {
			return fmt.Errorf("%s must be in key=value format: %q", flag, value)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("%s key cannot be empty: %q", flag, value)
		}
		dst[key] = val
	}
	return nil
}

func agentInstructions(bp *coop.Blueprint) string {
	preamble := fmt.Sprintf("You are building a production-grade Stripe integration: %q", bp.Title)
	return sessionLifecycleInstructions(preamble)
}

func guidedActionAgentInstructions(action *coop.GuidedAction) string {
	preamble := fmt.Sprintf("You are completing a guided co-op follow-up: %q.\n\n%s", action.Title, action.AgentContext)
	return sessionLifecycleInstructions(preamble)
}

func sessionLifecycleInstructions(preamble string) string {
	return fmt.Sprintf(`%s

Use context from the current app or codebase, if one exists, to inform your decisions. Inspect its architecture, language, framework, conventions, dependencies, and existing Stripe code so the integration fits the project naturally.

Work through one node at a time. Every start-work response includes an agent_prompt with the current task and acceptance criteria; do not work ahead. Write working code, run it, and report concrete file paths and test results. Use the latest Stripe SDK.

Before starting, run "stripe whoami". If you are not authenticated or it says "Test mode key: not available", run "stripe sandbox create --from-git"; the claim URL will appear in the developer's TUI.

For each node:
1. Run the next command returned by Co-op. Replace any <...> placeholders with real values before running it.
2. Follow that response's agent_prompt as the source of truth.
3. Verify the result and report each useful check with "stripe coop agent report-check".
4. Report the implementation with "stripe coop agent report-work".
5. Continue with the response's next command. If a task does not apply, use "stripe coop agent skip" with a reason.

Only await human review when the next command says to. Before awaiting, run the supplied review command, keep useful servers running, and give the developer concrete actions and expected results. Use a 5-minute shell timeout for await-review; re-run it if Co-op reports a timeout. If changes are requested, redo the affected node from the feedback. After the final confirmation, immediately run the returned next command.`, preamble)
}

func outputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func quoteArg(value string) string {
	return strconv.Quote(value)
}

func outputCoopError(msg, hint string) error {
	resp := coop.CommandResponse{
		OK:    false,
		Error: msg,
		Hint:  hint,
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintln(os.Stderr, string(data))
	return RenderedError{}
}

type RenderedError struct{}

func (RenderedError) Error() string {
	return "coop command failed"
}
