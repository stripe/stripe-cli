package coopcmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	coopskill "github.com/stripe/stripe-cli/pkg/coop/skill"
)

type coopRunCmd struct {
	cmd                   *cobra.Command
	language              string
	settings              []string
	agent                 string
	debugAgent            bool
	ensureSkill           func() error
	prepareSkillDiscovery func() error
	installCoopSkill      func(*harnessAdapter) (bool, error)
	// coopSkillReady reports whether the managed stripe-coop skill is
	// installed for the selected harness, so launch prompts may be compact.
	coopSkillReady bool
}

func newCoopRunCmd() *coopRunCmd {
	rc := &coopRunCmd{
		ensureSkill: ensureRepoStripeBestPracticesSkill,
		prepareSkillDiscovery: func() error {
			return ensureProjectSkillsDiscoveryRoot(claudeProjectDirectory)
		},
		installCoopSkill: ensureHarnessCoopSkill,
	}
	rc.cmd = &cobra.Command{
		Use:   "start [blueprint-id]",
		Short: "Launch a co-op session with an AI agent in split-screen",
		Long: `Starts a co-op session and launches an AI agent (Claude Code) in a
split terminal view. The TUI shows on one side, the agent works on the other.

If no blueprint is provided, the agent will explore your codebase,
understand your needs, and pick the right integration via the recommender.

Requires tmux (detected automatically). If already in a tmux session,
splits the current window. Otherwise creates a new tmux session.`,
		Example: `  stripe coop start
  stripe coop start one-time-payment --language=node
  stripe coop start --language=python`,
		Args: cobra.MaximumNArgs(1),
		RunE: rc.runCmd,
	}

	rc.cmd.Flags().StringVar(&rc.language, "language", "", "Programming language for the integration")
	rc.cmd.Flags().StringArrayVar(&rc.settings, "setting", nil, "Blueprint settings as key=value pairs")
	rc.cmd.Flags().StringVar(&rc.agent, "agent", "", "Agent to use (default: auto-detect claude/codex)")
	rc.cmd.Flags().BoolVar(&rc.debugAgent, "debug-agent", false, "Use a deterministic fake agent for local TUI debugging")
	mustMarkFlagHidden(rc.cmd, "debug-agent")

	return rc
}

func (rc *coopRunCmd) runCmd(cmd *cobra.Command, args []string) error {
	hasTmux := rc.hasTmux()
	inTmux := os.Getenv("TMUX") != ""

	var blueprintID string
	var blueprint *coop.Blueprint
	if len(args) > 0 {
		blueprintID = args[0]
		ctx := context.Background()
		if cmd != nil {
			ctx = cmd.Context()
		}
		var err error
		blueprint, err = coop.LoadBlueprint(ctx, coopBlueprintRepository(), blueprintID)
		if err != nil {
			return fmt.Errorf("%w. Run 'stripe coop recommend --all' to see available blueprints", err)
		}
		blueprintID = blueprint.Key
	}
	if rc.debugAgent && blueprintID == "" {
		return fmt.Errorf("--debug-agent requires a blueprint ID, e.g. stripe coop start one-time-payment --debug-agent")
	}
	stripeBin, _ := os.Executable()
	if rc.debugAgent {
		if err := rc.ensureStripeSkill(); err != nil {
			warnRepoStripeBestPracticesSkill(cmd, err)
		}
		buildDebugPane := rc.debugAgentPaneCommandBuilder(stripeBin)
		if inTmux {
			return rc.runInTmuxSplitWithCommand(stripeBin, blueprint, buildDebugPane)
		} else if hasTmux {
			return rc.runInNewTmuxWithCommand(stripeBin, blueprint, buildDebugPane)
		}
		return rc.runFallbackWithCommand(stripeBin, blueprint, buildDebugPane)
	}

	agent, err := rc.detectAgent()
	if err != nil {
		return err
	}
	if blueprintID == "" && !agent.adapter.interactiveLaunch {
		return fmt.Errorf(
			"%s cannot run the conversational blueprint discovery flow (%s)\n  Pick a blueprint first: stripe coop recommend --all\n  Then run: stripe coop start <blueprint-id>",
			agent.adapter.displayName, agent.adapter.notes,
		)
	}

	autoApprove, err := rc.promptAutoApprove(agent)
	if err != nil {
		return err
	}
	rc.coopSkillReady = rc.ensureCoopSkillFor(cmd, agent)
	if blueprintID != "" {
		if err := rc.ensureStripeSkill(); err != nil {
			warnRepoStripeBestPracticesSkill(cmd, err)
		}
	} else if agent.adapter.id == "claude" {
		if err := rc.prepareAgentSkillDiscovery(); err != nil {
			warnRepoClaudeSkillsDiscovery(cmd, err)
		}
	}
	if note := agent.adapter.notes; note != "" {
		var out io.Writer = os.Stderr
		if cmd != nil {
			out = cmd.ErrOrStderr()
		}
		fmt.Fprintf(out, "Note: %s\n", note)
	}
	fmt.Println()

	agentPrompt := rc.buildAgentPrompt(agent, blueprintID)

	if inTmux {
		return rc.runInTmuxSplit(stripeBin, agent, agentPrompt, autoApprove, blueprint)
	} else if hasTmux {
		return rc.runInNewTmux(stripeBin, agent, agentPrompt, autoApprove, blueprint)
	}

	return rc.runFallback(stripeBin, agent, agentPrompt, autoApprove, blueprint)
}

func (rc *coopRunCmd) ensureStripeSkill() error {
	if rc.ensureSkill != nil {
		return rc.ensureSkill()
	}
	return ensureRepoStripeBestPracticesSkill()
}

func (rc *coopRunCmd) prepareAgentSkillDiscovery() error {
	if rc.prepareSkillDiscovery != nil {
		return rc.prepareSkillDiscovery()
	}
	return ensureProjectSkillsDiscoveryRoot(claudeProjectDirectory)
}

// ensureCoopSkillFor installs the bundled stripe-coop skill for the selected
// harness. Install failures degrade to the full self-contained launch prompt,
// so they warn instead of blocking the session.
func (rc *coopRunCmd) ensureCoopSkillFor(cmd *cobra.Command, agent *agentInfo) bool {
	install := rc.installCoopSkill
	if install == nil {
		install = ensureHarnessCoopSkill
	}
	installed, err := install(agent.adapter)
	if err != nil {
		var out io.Writer = os.Stderr
		if cmd != nil {
			out = cmd.ErrOrStderr()
		}
		if errors.Is(err, coopskill.ErrUnmanagedSkill) {
			fmt.Fprintf(out, "Warning: an existing %q skill not managed by the Stripe CLI was left untouched; launching with full instructions instead: %v\n", coopskill.Name, err)
		} else {
			fmt.Fprintf(out, "Warning: unable to install the %q skill; launching with full instructions instead: %v\n", coopskill.Name, err)
		}
		return false
	}
	return installed
}

func (rc *coopRunCmd) buildAgentPrompt(agent *agentInfo, blueprintID string) string {
	if blueprintID != "" {
		return ""
	}
	if rc.coopSkillReady {
		return rc.buildCompactDiscoveryPrompt(agent)
	}

	langHint := ""
	if rc.language != "" {
		langHint = fmt.Sprintf("\nThe developer is working in %s.", rc.language)
	}

	return fmt.Sprintf(`You are helping a developer build a production-grade Stripe integration.

A developer is watching your progress in a live terminal UI (the other pane).%s

%s

Use context from the current app or codebase, if one exists, to inform your recommendations and decisions. Inspect its architecture, language, framework, conventions, dependencies, and existing Stripe code so the integration fits the project naturally.

Your first job is to understand what they're building and what they need from Stripe. Do NOT assume they know Stripe product names.

Steps:
1. Look at the codebase to understand the language, framework, and what the project does.
2. Based on what you find:
   IF code exists: Summarize what the project does in 1-2 sentences and ask the developer
   to confirm. Then ask: "What would you like to build with Stripe?"
   IF no code exists (empty project): Ask "What are you looking to build?"
   WAIT for their response. Do NOT proceed until they answer.
   Do NOT assume what they need. Let them tell you in their own words.
3. Based on their answer, run "stripe coop recommend --all" and pick the best blueprint from the returned summaries.
4. Explain what you found in simple terms: "I'll set up X which lets you do Y" and confirm.
5. Only after confirmation, run "stripe coop run <blueprint-id> --language=<lang>".
6. Follow the instructions in the JSON response and work through each step.

The developer will confirm each step in the TUI before you proceed.

Important: Run "stripe whoami" first to check auth. If not logged in OR if it shows "Test mode key: not available", run "stripe sandbox create --from-git" to provision a sandbox. The claim URL will appear automatically in the TUI.

%s`, langHint, coopAgentCoordinationInstructions(), stripeAgentGuidanceInstructions())
}

func (rc *coopRunCmd) buildAgentPromptForSession(agent *agentInfo, session *coop.Session) (string, error) {
	title := session.Blueprint
	if session.BlueprintPin != nil && session.BlueprintPin.Title != "" {
		title = session.BlueprintPin.Title
	}
	resp := newCoopAgentSessionResponse(
		title,
		session,
		sessionLifecycleInstructions(fmt.Sprintf("You are building a production-grade Stripe integration: %q", title)),
	)
	if rc.coopSkillReady && agent != nil {
		return rc.buildCompactSessionPrompt(agent, session, title, resp.Next), nil
	}

	return fmt.Sprintf(`You are running a Stripe co-op integration session. A developer is watching your progress in a live terminal UI.

%s

The session is already created. After the authentication check above, begin by running this command exactly:

%s

Continue using the agent_prompt and next fields returned by the typed Co-op commands.`, resp.AgentPrompt, resp.Next), nil
}

// buildCompactSessionPrompt relies on the installed stripe-coop skill for the
// stable lifecycle and safety contract, so the launch prompt carries only the
// skill activation and this session's dynamic context.
func (rc *coopRunCmd) buildCompactSessionPrompt(agent *agentInfo, session *coop.Session, title, next string) string {
	var prompt strings.Builder
	fmt.Fprintf(&prompt, "%s\n\n", agent.adapter.activationLine)
	fmt.Fprintf(&prompt, "Session: %s\n", session.ID)
	fmt.Fprintf(&prompt, "Integration: %s\n", title)
	if language := session.Settings["language"]; language != "" {
		fmt.Fprintf(&prompt, "Language: %s\n", language)
	}
	prompt.WriteString("A developer reviews your work live in the Co-op TUI pane.\n\n")
	fmt.Fprintf(&prompt, "Begin by running exactly:\n%s\n\n", next)
	prompt.WriteString("If the stripe-coop skill is unavailable, run the command anyway and follow each response's agent_prompt, next, and recovery fields.")
	return prompt.String()
}

// buildCompactDiscoveryPrompt is the skill-backed no-blueprint launch prompt.
func (rc *coopRunCmd) buildCompactDiscoveryPrompt(agent *agentInfo) string {
	var prompt strings.Builder
	fmt.Fprintf(&prompt, "%s\n\n", agent.adapter.activationLine)
	prompt.WriteString("No blueprint is selected. Inspect the project and gather only developer intent that cannot be inferred, then follow the skill's discovery flow starting with `stripe coop recommend --all`.\n")
	if rc.language != "" {
		fmt.Fprintf(&prompt, "Language hint: %s.\n", rc.language)
	}
	prompt.WriteString("A developer reviews your work live in the Co-op TUI pane.")
	return prompt.String()
}

func (rc *coopRunCmd) startSessionQuietly(blueprint *coop.Blueprint) (*coop.Session, error) {
	if blueprint == nil {
		return nil, fmt.Errorf("cannot start a session without a blueprint")
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return nil, err
	}

	sessionID := "coop_" + generateShortID()
	session, err := newCoopSession(blueprint, sessionID, rc.language, rc.settings, nil, "", "")
	if err != nil {
		return nil, err
	}
	if err := store.Write(session); err != nil {
		return nil, err
	}
	return session, nil
}

func (rc *coopRunCmd) abortStartedSession(session *coop.Session, note string) {
	if session == nil {
		return
	}
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return
	}
	_, _ = store.Update(session.ID, func(session *coop.Session) error {
		session.Status = coop.SessionAborted
		if note != "" {
			node, _ := session.ActiveNode()
			if node == nil {
				node, _ = session.NodeByNumber(1)
			}
			if node != nil {
				node.Activity = note
			}
		}
		return nil
	})
}
