package coopcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

type coopAgentRunCmd struct {
	cmd           *cobra.Command
	language      string
	settings      []string
	params        []string
	parentSession string
	parentStep    string
	ensureSkill   func() error
}

func newCoopAgentRunCmd() *coopAgentRunCmd {
	rc := &coopAgentRunCmd{ensureSkill: ensureRepoStripeBestPracticesSkill}
	rc.cmd = &cobra.Command{
		Use:   "run <blueprint-id>",
		Short: "Create a co-op session from a blueprint (agent-facing)",
		Long: `Creates a new co-op session using the specified blueprint. The session file
is written to disk and the agent can immediately begin working through nodes.

This is the agent-facing command. Developers should use "stripe coop start" instead.`,
		Example: `  stripe coop run one-time-payment
  stripe coop run one-time-payment --language=node
  stripe coop run setup-future-payments --setting=framework=express --param=customer_type=existing`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return nil
			}
			return outputCoopError(
				"coop run requires exactly one blueprint ID",
				"Choose a blueprint before starting a session.",
				coop.Continue("stripe coop recommend"),
			)
		},
		RunE: rc.runCmd,
	}

	rc.cmd.Flags().StringVar(&rc.language, "language", "", "Programming language for the integration")
	rc.cmd.Flags().StringArrayVar(&rc.settings, "setting", nil, "Blueprint settings as key=value pairs")
	rc.cmd.Flags().StringArrayVar(&rc.params, "param", nil, "Blueprint params as key=value pairs")
	rc.cmd.Flags().StringVar(&rc.parentSession, "parent-session", "", "Parent co-op session ID for follow-up work")
	rc.cmd.Flags().StringVar(&rc.parentStep, "parent-step", "", "Parent next-step ID this session fulfills")
	configureAgentCommand(rc.cmd)

	return rc
}

func (rc *coopAgentRunCmd) runCmd(cmd *cobra.Command, args []string) error {
	blueprintID := args[0]
	bp, err := coop.LoadBlueprint(cmd.Context(), coopBlueprintRepository(), blueprintID)
	if err != nil {
		// Surface the specific error (e.g. an ambiguous prefix and its candidate
		// list) rather than a generic "not found".
		return outputCoopError(
			err.Error(),
			"Choose an available blueprint ID.",
			coop.Continue("stripe coop recommend --all"),
		)
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return outputAgentError(fmt.Errorf("creating store: %w", err))
	}

	sessionID := "coop_" + uuid.New().String()[:8]

	session, err := newCoopSession(bp, sessionID, rc.language, rc.settings, rc.params, rc.parentSession, rc.parentStep)
	if err != nil {
		return outputCoopError(
			err.Error(),
			"Retry without the malformed setting or parameter; optional values use key=value syntax.",
			coop.Continue(coop.RunCommand(blueprintID)),
		)
	}
	if err := rc.ensureStripeSkill(); err != nil {
		warnRepoStripeBestPracticesSkill(cmd, err)
	}

	if err := store.Write(session); err != nil {
		return outputAgentError(fmt.Errorf("writing session: %w", err))
	}

	resp := newCoopAgentRunResponse(bp, session)

	return outputJSON(resp)
}

func (rc *coopAgentRunCmd) ensureStripeSkill() error {
	if rc.ensureSkill != nil {
		return rc.ensureSkill()
	}
	return ensureRepoStripeBestPracticesSkill()
}

func newCoopAgentRunResponse(bp *coop.Blueprint, session *coop.Session) coop.CommandResponse {
	return newCoopAgentSessionResponse(bp.Title.DefaultMessage, session, agentInstructions(bp))
}

func newCoopAgentGuidedActionResponse(action *coop.GuidedAction, session *coop.Session) coop.CommandResponse {
	return newCoopAgentSessionResponse(action.Title, session, guidedActionAgentInstructions(action))
}

func newCoopAgentSessionResponse(title string, session *coop.Session, instructions string) coop.CommandResponse {
	return coop.CommandResponse{
		OK:           true,
		SessionID:    session.ID,
		Node:         1,
		State:        "created",
		Message:      fmt.Sprintf("Session started: %s (%d nodes)", title, session.TotalNodes()),
		Continuation: coop.Continue(coop.StartWorkCommand(session.ID, 1, "Beginning: "+session.Steps[0].Nodes[0].TitleText())),
		AgentPrompt:  instructions,
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

	session, err := coop.NewSessionFromBlueprint(bp, sessionID, settings, params)
	if err != nil {
		return nil, err
	}
	session.CreatedAt = time.Now().UTC()
	if cwd, err := os.Getwd(); err == nil {
		session.Cwd = cwd
	}
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
	preamble := fmt.Sprintf("You are building a production-grade Stripe integration: %q", bp.Title.DefaultMessage)
	return sessionLifecycleInstructions(preamble)
}

func guidedActionAgentInstructions(action *coop.GuidedAction) string {
	preamble := fmt.Sprintf("You are completing a guided co-op follow-up: %q.\n\n%s", action.Title, action.AgentContext)
	return sessionLifecycleInstructions(preamble)
}

func sessionLifecycleInstructions(preamble string) string {
	return fmt.Sprintf(`%s

The blueprint describes the Stripe flow the developer wants in their app. Your deliverable is the user's app implementing that flow. Stripe CLI commands are useful for setup and verification, but they are not the implementation unless a node is explicitly a cliCommand.

%s

BEFORE YOU START — ensure you have API access:
1. Run "stripe whoami" to check if you're authenticated.
2. If not authenticated OR if the output shows "Test mode key: not available",
   run "stripe sandbox create --from-git" to provision a sandbox.
   This gives you a working API key without requiring browser login.
   The claim URL will appear automatically in the TUI for the developer.

Security and configuration:
- Never hardcode secret or restricted API keys, webhook secrets, or Stripe-shaped fake secrets in source code, tests, examples, or fallback paths.
- Application code must read secrets from environment variables or the app's existing secret-management pattern. Stripe CLI commands may use the authenticated CLI config. If application credentials are unavailable, return a clear setup error instead of adding a placeholder key.
- Supply publishable keys through the app's normal public client configuration or build environment instead of scattering literals through the codebase.
- In tests, use non-secret fixture strings such as "test_webhook_secret" rather than values that look like real Stripe secrets.

Each node has a description that tells you what to do. Follow the description — it's the source of truth. The node type defines the expected kind of app integration:
- "apiRequest": Implement app code that calls this Stripe API using the official Stripe SDK or the project's existing Stripe client pattern. Use the Stripe CLI only to inspect, create temporary test data, or verify the app code.
- "asyncHandler": Implement the app's webhook or async event handler for every event listed on the node. Read the raw request body, verify the Stripe signature with STRIPE_WEBHOOK_SECRET using the official SDK webhook helpers, branch on each listed event type, and store or act on the event data the app needs. Use "stripe listen --forward-to localhost:<actual app port>/webhook" and "stripe trigger <event>" when that event has a supported trigger; otherwise exercise the app/API flow or test helper that emits it. Do not hardcode port 4242 unless the app is actually listening there.
- "uiComponent": Build or update the user-facing app surface that starts, redirects to, or displays this part of the flow. Verify it through the app.
- "cliCommand": Run a CLI command (e.g. stripe projects init, stripe projects deploy). This is the only node type where no app code may be required.
- "testHelper": Verify the app behavior end-to-end. Use Stripe test helpers, test clocks, triggers, or CLI commands as supporting test tools.

For apiRequest, asyncHandler, and uiComponent nodes, a node is complete only when:
1. The user's app has working code for the behavior, whether newly implemented or already present, unless the node truly does not apply.
2. The code is wired into the app's existing route, service, handler, UI, or framework conventions.
3. Verification exercises the app code, not only a direct Stripe CLI/API call.
4. report-work points to the relevant app file/function/route you implemented, changed, or verified. Do not report README/package files as the main implementation unless the node is documentation-only.

Run at least one meaningful report-check before report-work for every non-skipped reviewable node, and add --passed only after observing the expected result. If the environment prevents full verification, report the concrete partial or failed check without --passed and explain the exact limitation instead of claiming success.

%s

Use context from the current app or codebase, if one exists, to inform your decisions. Inspect its architecture, language, framework, conventions, dependencies, and existing Stripe code so the integration fits the project naturally.

Work through one node at a time. Every start-work response includes an agent_prompt with the current task and acceptance criteria; do not work ahead. Write working code, run it, and report concrete file paths and test results. Use the latest Stripe SDK.

Before starting, run "stripe whoami". If you are not authenticated or it says "Test mode key: not available", run "stripe sandbox create --from-git"; the claim URL will appear in the developer's TUI.

For each node:
1. Run the next command returned by Co-op. Replace any <...> placeholders with real values before running it.
2. Follow that response's agent_prompt as the source of truth.
3. Verify the result and report each useful check with "stripe coop agent report-check".
4. Report the implementation with "stripe coop agent report-work".
5. Continue with the response's next command. If a task does not apply, use "stripe coop agent skip" with a reason.

Only await human review when the next command says to. Before awaiting, run the supplied review command, keep useful servers running, and give the developer concrete actions and expected results. Co-op returns from await-review after %s; allow the shell command at least %s so the structured timeout response arrives. Re-run it if Co-op reports a timeout. If Co-op wakes you after later human input, run the exact stripe coop agent resume command it provides, then run a non-empty next command exactly; an empty next means another handoff already advanced the session. If changes are requested, redo the affected task from the feedback. After the final confirmation, immediately run the returned next command.

Never pass full card numbers to Stripe APIs or CLI commands. Collect card details only through hosted Checkout, Payment Element, or another official client-side integration. If an API needs a test payment method, use a supported test PaymentMethod ID such as pm_card_visa.`,
		preamble,
		coopAgentCoordinationInstructions(),
		stripeAgentGuidanceInstructions(),
		workflow.AwaitTimeout,
		workflow.AwaitHarnessTimeout,
	)
}

func outputJSON(v interface{}) error {
	return outputJSONTo(os.Stdout, v)
}

func outputJSONTo(w io.Writer, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

type RenderedError struct{}

func (RenderedError) Error() string {
	return "coop command failed"
}
