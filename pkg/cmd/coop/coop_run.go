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
		return outputCoopError(fmt.Sprintf("Blueprint %q not found. Run 'stripe coop recommend' to see available blueprints.", blueprintID), "stripe coop recommend")
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

	// Build node overview for the agent
	var nodes []nodeBrief
	nodeNumber := 0
	for _, step := range session.Steps {
		for _, n := range step.Nodes {
			nodeNumber++
			nodes = append(nodes, nodeBrief{
				Number:        nodeNumber,
				Title:         n.Title,
				Type:          string(n.Type),
				Description:   n.Description,
				ReviewPrompt:  n.ReviewPrompt,
				ReviewCommand: n.ReviewCommand,
				AutoConfirm:   n.AutoConfirm,
			})
		}
	}

	resp := coopAgentRunResponse{
		CommandResponse: coop.CommandResponse{
			OK:        true,
			SessionID: sessionID,
			Node:      1,
			State:     "created",
			Message:   fmt.Sprintf("Session started: %s (%d nodes)", bp.Title, session.TotalNodes()),
			Next:      fmt.Sprintf("stripe coop agent start-work --session=%s --step=1 --note=%s", sessionID, quoteArg("Beginning: "+session.Steps[0].Nodes[0].Title)),
		},
		AgentInstructions: agentInstructions(bp, session),
		Nodes:             nodes,
	}

	return outputJSON(resp)
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

type coopAgentRunResponse struct {
	coop.CommandResponse
	AgentInstructions string      `json:"agent_instructions"`
	Nodes             []nodeBrief `json:"nodes"`
}

type nodeBrief struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	Type          string `json:"type"`
	Description   string `json:"description,omitempty"`
	ReviewPrompt  string `json:"review_prompt,omitempty"`
	ReviewCommand string `json:"review_command,omitempty"`
	AutoConfirm   bool   `json:"auto_confirm,omitempty"`
}

func agentInstructions(bp *coop.Blueprint, session *coop.Session) string {
	preamble := fmt.Sprintf("You are building a working Stripe integration: %q", bp.Title)

	return fmt.Sprintf(`%s

BEFORE YOU START — ensure you have API access:
1. Run "stripe whoami" to check if you're authenticated.
2. If not authenticated OR if the output shows "Test mode key: not available",
   run "stripe sandbox create --from-git" to provision a sandbox.
   This gives you a working API key without requiring browser login.
   The claim URL will appear automatically in the TUI for the developer.

Each node has a description that tells you what to do. Follow the description — it's the source of truth. The node type is a hint about the general category:
- "apiRequest": Usually means writing code that calls a Stripe API. Run it and verify the response.
- "asyncHandler": Set up a webhook handler. Use "stripe listen --forward-to localhost:<port>/webhook" to test.
- "uiComponent": Build frontend code or configure something user-facing. Verify it works.
- "cliCommand": Run a CLI command (e.g. stripe projects init, stripe projects deploy). Report the output.
- "testHelper": Verify something works end-to-end. Run the flow and confirm the expected outcome.

If a node includes review_prompt, that is the baseline acceptance check shown to the human. If it includes review_command, run that exact command when verifying or explain why it does not apply. Make your implementation note and verifications directly answer these fields. When you add verification checks, write them as useful confirmation guidance for the human too: include concrete actions and expected results, such as "Visit http://localhost:3000/checkout, click Pay, and confirm the browser redirects to Stripe Checkout" rather than vague labels like "manual test passed".

Node 1 is always "Understand the project" — scan files, identify the tech stack, and summarize what you found. This helps you adapt the remaining nodes to the developer's actual setup. Don't ask the developer questions you can answer by reading the code.

Agent lifecycle commands (use this session id: %s):
1. stripe coop agent start-work --session=%s --step=<n> --note="<what you're about to do>"
2. Write the code and run it to verify it works
3. stripe coop agent report-check --session=%s --step=<n> --check="<what you verified>" --passed
4. stripe coop agent report-work --session=%s --step=<n> --file=<main file> --lines=<range> --snippet="<key code>" --note="<summary>"
5. Follow the JSON response's next command. Most nodes continue to the next node in the same step.
6. Only run stripe coop agent await-review --session=%s --step=<n> when the response says the step is ready for review. Await blocks until the human confirms the step or requests changes.
7. If confirmed: move to next node. If rejected: redo the affected node (check the message for feedback).
8. When the final node is confirmed: IMMEDIATELY run "stripe coop agent next-action --session=%s". Do not stop or ask — just run it. It shows the developer their options in the TUI and blocks until they choose.

Steps are the default human-review unit. Build and verify each node one at a time, but do not interrupt the developer for every node. At the end of each step, before running await, help the developer verify the step: run relevant review_command values, start any needed app/server, keep useful processes running, share the local URL or command to open it, create or identify test data, and explain exactly what observable result they should confirm. Add these concrete user-facing checks with stripe coop agent report-check --session=%s --step=<n> --check="..." --passed so the review card has useful evidence.

The "await" command is critical at step boundaries — it blocks until the developer acts. Do NOT proceed to the next step without running await when the node response tells you the step is ready. Set a 5-minute timeout on the shell command (it will re-prompt you if it times out). If changes are requested, ask the developer what they'd like you to change before redoing the affected node.

Some nodes are marked auto_confirm — these do not require human review. Continue following the next command returned by the CLI.

Important:
- The human is watching your progress live in a terminal UI.
- Write working code, not stubs. Run it. Verify it actually works.
- Report what you did concretely (file paths, line numbers, test results).
- If a node doesn't apply to the user's setup, skip it: stripe coop agent skip --session=%s --step=<n> --note="<reason>"
- Always install the LATEST version of the Stripe SDK for the language in use. Do not pin to old versions.
  Examples: "npm install stripe@latest", "pip install --upgrade stripe", "gem install stripe"
  Check https://docs.stripe.com/libraries for current versions if unsure.`, preamble, session.ID, session.ID, session.ID, session.ID, session.ID, session.ID, session.ID, session.ID)
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
