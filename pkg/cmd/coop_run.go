package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopStartCmd struct {
	cmd           *cobra.Command
	language      string
	settings      []string
	parentSession string
	parentStep    string
}

func newCoopStartCmd() *coopStartCmd {
	sc := &coopStartCmd{}
	sc.cmd = &cobra.Command{
		Use:   "run <blueprint-id>",
		Short: "Create a co-op session from a blueprint (agent-facing)",
		Long: `Creates a new co-op session using the specified blueprint. The session file
is written to disk and the agent can immediately begin working through steps.

This is the agent-facing command. Developers should use "stripe coop start" instead.`,
		Example: `  stripe coop run one-time-payment
  stripe coop run one-time-payment --language=node
  stripe coop run setup-future-payments --setting=framework=express`,
		Args: cobra.ExactArgs(1),
		RunE: sc.runStartCmd,
	}

	sc.cmd.Flags().StringVar(&sc.language, "language", "", "Programming language for the integration")
	sc.cmd.Flags().StringArrayVar(&sc.settings, "setting", nil, "Blueprint settings as key=value pairs")
	sc.cmd.Flags().StringVar(&sc.parentSession, "parent-session", "", "Parent co-op session ID for follow-up work")
	sc.cmd.Flags().StringVar(&sc.parentStep, "parent-step", "", "Parent next-step ID this session fulfills")

	return sc
}

func (sc *coopStartCmd) runStartCmd(cmd *cobra.Command, args []string) error {
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

	session := sc.newSession(bp, sessionID)
	session.CreatedAt = time.Now().UTC()

	// Populate claim URL from config if sandbox was provisioned
	if claimURL := Config.Profile.GetConfigField("sandbox_claim_url"); claimURL != "" {
		session.ClaimURL = viper.GetString(Config.Profile.GetConfigField("sandbox_claim_url"))
	}

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	// Build step overview for the agent
	var steps []stepBrief
	stepNum := 0
	for _, ch := range session.Chapters {
		for _, n := range ch.Nodes {
			stepNum++
			steps = append(steps, stepBrief{
				Number:            stepNum,
				Title:             n.Title,
				Type:              string(n.Type),
				Description:       n.Description,
				ReviewPrompt:      n.ReviewPrompt,
				ReviewCommand:     n.ReviewCommand,
				ReviewGranularity: string(ch.ReviewGranularity),
				AutoConfirm:       n.AutoConfirm,
			})
		}
	}

	resp := coopStartResponse{
		CommandResponse: coop.CommandResponse{
			OK:        true,
			SessionID: sessionID,
			Step:      1,
			State:     "created",
			Message:   fmt.Sprintf("Session started: %s (%d steps)", bp.Title, session.TotalSteps()),
			Next:      fmt.Sprintf("stripe coop step 1 start --session=%s --note=%s", sessionID, quoteArg("Beginning: "+session.Chapters[0].Nodes[0].Title)),
		},
		AgentInstructions: agentInstructions(bp, session),
		Steps:             steps,
	}

	return outputJSON(resp)
}

func (sc *coopStartCmd) newSession(bp *coop.Blueprint, sessionID string) *coop.Session {
	settings := make(map[string]string)
	if sc.language != "" {
		settings["language"] = sc.language
	}
	for _, s := range sc.settings {
		for i := range s {
			if s[i] == '=' {
				settings[s[:i]] = s[i+1:]
				break
			}
		}
	}

	session := coop.NewSessionFromBlueprint(bp, sessionID, settings)
	session.ParentSessionID = sc.parentSession
	session.ParentStepID = sc.parentStep
	return session
}

type coopStartResponse struct {
	coop.CommandResponse
	AgentInstructions string      `json:"agent_instructions"`
	Steps             []stepBrief `json:"steps"`
}

type stepBrief struct {
	Number            int    `json:"number"`
	Title             string `json:"title"`
	Type              string `json:"type"`
	Description       string `json:"description,omitempty"`
	ReviewPrompt      string `json:"review_prompt,omitempty"`
	ReviewCommand     string `json:"review_command,omitempty"`
	ReviewGranularity string `json:"review_granularity,omitempty"`
	AutoConfirm       bool   `json:"auto_confirm,omitempty"`
}

func agentInstructions(bp *coop.Blueprint, session *coop.Session) string {
	preamble := fmt.Sprintf("You are building a working Stripe integration: %q", bp.Title)
	if bp.Prompt != "" {
		preamble = bp.Prompt
	}

	return fmt.Sprintf(`%s

BEFORE YOU START — ensure you have API access:
1. Run "stripe whoami" to check if you're authenticated.
2. If not authenticated OR if the output shows "Test mode key: not available",
   run "stripe sandbox create --from-git" to provision a sandbox.
   This gives you a working API key without requiring browser login.
   The claim URL will appear automatically in the TUI for the developer.

Each step has a description that tells you what to do. Follow the description — it's the source of truth. The node type is a hint about the general category:
- "apiRequest": Usually means writing code that calls a Stripe API. Run it and verify the response.
- "asyncHandler": Set up a webhook handler. Use "stripe listen --forward-to localhost:<port>/webhook" to test.
- "uiComponent": Build frontend code or configure something user-facing. Verify it works.
- "cliCommand": Run a CLI command (e.g. stripe projects init, stripe projects deploy). Report the output.
- "testHelper": Verify something works end-to-end. Run the flow and confirm the expected outcome.

If a step includes review_prompt, that is the baseline acceptance check shown to the human. If it includes review_command, run that exact command when verifying or explain why it does not apply. Make your implementation note and verifications directly answer these fields. When you add verification checks, write them as useful confirmation guidance for the human too: include concrete actions and expected results, such as "Visit http://localhost:3000/checkout, click Pay, and confirm the browser redirects to Stripe Checkout" rather than vague labels like "manual test passed".

Step 1 is always "Understand the project" — scan files, identify the tech stack, and summarize what you found. This helps you adapt the remaining steps to the developer's actual setup. Don't ask the developer questions you can answer by reading the code.

Step lifecycle commands (use this session id: %s):
1. stripe coop step <n> start --session=%s --note="<what you're about to do>"
2. Write the code and run it to verify it works
3. stripe coop step <n> verify --session=%s --check="<what you verified>" --passed
4. stripe coop step <n> done --session=%s --file=<main file> --lines=<range> --snippet="<key code>" --note="<summary>"
5. Follow the JSON response's next command. Most steps continue to the next step in the same chapter.
6. Only run stripe coop step <n> await --session=%s when the response says the chapter is ready for review. Await blocks until the human confirms the chapter or requests changes.
7. If confirmed: move to next step. If rejected: redo the affected step (check the message for feedback).
8. When the final step is confirmed: IMMEDIATELY run "stripe coop next-steps --session=%s". Do not stop or ask — just run it. It shows the developer their options in the TUI and blocks until they choose.

Chapters are the default human-review unit. Build and verify each step one at a time, but do not interrupt the developer for every step. At the end of each chapter, before running await, help the developer verify the chapter: run relevant review_command values, start any needed app/server, keep useful processes running, share the local URL or command to open it, create or identify test data, and explain exactly what observable result they should confirm. Add these concrete user-facing checks with stripe coop step <n> verify --check="..." --passed so the review card has useful evidence.

The "await" command is critical at chapter boundaries — it blocks until the developer acts. Do NOT proceed to the next chapter without running await when the step response tells you the chapter is ready. Set a 5-minute timeout on the shell command (it will re-prompt you if it times out). If changes are requested, ask the developer what they'd like you to change before redoing the affected step.

Some steps are marked auto_confirm — these do not require human review. Continue following the next command returned by the CLI.

For values containing special characters ($, quotes, etc.), use --stdin to pipe JSON:
  echo '{"file":"app.js","lines":"1-10","snippet":"code here","note":"Created $99 product"}' | stripe coop step <n> done --session=%s --stdin

Important:
- The human is watching your progress live in a terminal UI.
- Write working code, not stubs. Run it. Verify it actually works.
- Report what you did concretely (file paths, line numbers, test results).
- If a step doesn't apply to the user's setup, skip it: stripe coop step <n> skip --note="<reason>"
- Always install the LATEST version of the Stripe SDK for the language in use. Do not pin to old versions.
  Examples: "npm install stripe@latest", "pip install --upgrade stripe", "gem install stripe"
  Check https://docs.stripe.com/libraries for current versions if unsure.`, preamble, session.ID, session.ID, session.ID, session.ID, session.ID, session.ID, session.ID)
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
	return coopRenderedError{}
}

type coopRenderedError struct{}

func (coopRenderedError) Error() string {
	return "coop command failed"
}
