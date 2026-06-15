package coopcmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopRunCmd struct {
	cmd        *cobra.Command
	language   string
	settings   []string
	agent      string
	debugAgent bool
}

func newCoopRunCmd() *coopRunCmd {
	rc := &coopRunCmd{}
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
	if len(args) > 0 {
		blueprintID = args[0]
		if _, err := coop.LoadBlueprint(blueprintID); err != nil {
			return fmt.Errorf("blueprint %q not found. Run 'stripe coop recommend' to see available blueprints", blueprintID)
		}
	}

	stripeBin, _ := os.Executable()
	if rc.debugAgent {
		if blueprintID == "" {
			return fmt.Errorf("--debug-agent requires a blueprint ID, e.g. stripe coop start one-time-payment --debug-agent")
		}
		buildDebugPane := rc.debugAgentPaneCommandBuilder(stripeBin)
		if inTmux {
			return rc.runInTmuxSplitWithCommand(stripeBin, blueprintID, buildDebugPane)
		} else if hasTmux {
			return rc.runInNewTmuxWithCommand(stripeBin, blueprintID, buildDebugPane)
		}
		return rc.runFallbackWithCommand(stripeBin, blueprintID, buildDebugPane)
	}

	agent, err := rc.detectAgent()
	if err != nil {
		return err
	}

	autoApprove := rc.promptAutoApprove(agent)
	fmt.Println()

	agentPrompt := rc.buildAgentPrompt(blueprintID)

	if inTmux {
		return rc.runInTmuxSplit(stripeBin, agent, agentPrompt, autoApprove, blueprintID)
	} else if hasTmux {
		return rc.runInNewTmux(stripeBin, agent, agentPrompt, autoApprove, blueprintID)
	}

	return rc.runFallback(stripeBin, agent, agentPrompt, autoApprove, blueprintID)
}

func (rc *coopRunCmd) buildAgentPrompt(blueprintID string) string {
	if blueprintID != "" {
		return ""
	}

	langHint := ""
	if rc.language != "" {
		langHint = fmt.Sprintf("\nThe developer is working in %s.", rc.language)
	}

	return fmt.Sprintf(`You are helping a developer add Stripe to their project.

A developer is watching your progress in a live terminal UI (the other pane).%s

Your first job is to understand what they're building and what they need from Stripe. Do NOT assume they know Stripe product names.

Steps:
1. Look at the codebase to understand the language, framework, and what the project does.
2. Based on what you find:
   IF code exists: Summarize what the project does in 1-2 sentences and ask the developer
   to confirm. Then ask: "What would you like to build with Stripe?"
   IF no code exists (empty project): Ask "What are you looking to build?"
   WAIT for their response. Do NOT proceed until they answer.
   Do NOT assume what they need. Let them tell you in their own words.
3. Based on their answer, run "stripe coop recommend --query=<description of what they need>"
4. Explain what you found in simple terms: "I'll set up X which lets you do Y" and confirm.
5. Only after confirmation, run "stripe coop run <blueprint-id> --language=<lang>".
6. Follow the instructions in the JSON response and work through each step.

The developer will confirm each step in the TUI before you proceed.

Important: Run "stripe whoami" first to check auth. If not logged in OR if it shows "Test mode key: not available", run "stripe sandbox create --from-git" to provision a sandbox. The claim URL will appear automatically in the TUI.`, langHint)
}

func (rc *coopRunCmd) buildAgentPromptForSession(session *coop.Session) (string, error) {
	bp, err := coop.LoadBlueprint(session.Blueprint)
	if err != nil {
		return "", err
	}
	resp := newCoopAgentRunResponse(bp, session)
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`You are running a Stripe co-op integration session.

A developer is watching your progress in a live terminal UI (the other pane).

The session is already created. Use this structured start response as the protocol source of truth:

%s

Start by running the "next" command exactly as written. Then follow agent_instructions and continue using the JSON responses from the typed co-op agent commands.

Important: Run "stripe whoami" first to check auth. If not logged in OR if it shows "Test mode key: not available", run "stripe sandbox create --from-git" to provision a sandbox. The claim URL will appear automatically in the TUI.`, string(data)), nil
}

func (rc *coopRunCmd) startSessionQuietly(blueprintID string) (*coop.Session, error) {
	bp, err := coop.LoadBlueprint(blueprintID)
	if err != nil {
		return nil, err
	}

	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return nil, err
	}

	sessionID := "coop_" + generateShortID()
	session, err := newCoopSession(bp, sessionID, rc.language, rc.settings, nil, "", "")
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
