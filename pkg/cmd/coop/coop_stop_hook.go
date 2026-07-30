package coopcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/workflow"
)

// maxConsecutiveStopBlocks bounds how many times in a row the hook holds a turn
// open without the session advancing. Blocking forever would burn tokens in a
// tight loop whenever the developer walks away mid-review, which is a worse
// failure than the drift this hook prevents. After the limit the agent may
// stop; its heartbeat goes stale and the TUI's existing idle state tells the
// developer to rejoin.
const maxConsecutiveStopBlocks = 3

// staleSessionFallbackAge bounds the discovery-mode fallback: an untouched
// session older than this is assumed to belong to some other run.
const staleSessionFallbackAge = 30 * time.Minute

// stopHookStore is the slice of coop.Store the hook needs, so tests can supply
// a session without a config directory.
type stopHookStore interface {
	Read(id string) (*coop.Session, error)
	ActiveSessions() ([]*coop.Session, error)
	ReadStopHookState(id string) (coop.StopHookState, error)
	WriteStopHookState(id string, state coop.StopHookState) error
	RemoveStopHookState(id string) error
}

// stopHookDecision is the JSON contract shared by Claude Code and Codex Stop
// hooks: decision "block" feeds Reason back to the agent as its next
// instruction instead of letting the turn end.
type stopHookDecision struct {
	Decision      string `json:"decision,omitempty"`
	Reason        string `json:"reason,omitempty"`
	SystemMessage string `json:"systemMessage,omitempty"`
}

type coopStopHookCmd struct {
	cmd       *cobra.Command
	session   string
	configDir string
}

func newCoopStopHookCmd() *coopStopHookCmd {
	c := &coopStopHookCmd{}
	c.cmd = &cobra.Command{
		Use:    "stop-hook",
		Short:  "Agent harness Stop hook: hold the turn while Co-op has work pending",
		Hidden: true,
		Long: `Reads an agent harness's Stop event on stdin and decides whether the agent
may end its turn. While the session still has an actionable next command, the
hook blocks and hands that exact command back, so a model that would otherwise
drift out of the Co-op lifecycle is pulled into it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Drain first: every return path below must leave the harness's
			// event payload consumed, or it can see a broken pipe. Nothing in
			// the payload is needed — the session file has everything.
			_, _ = io.Copy(io.Discard, cmd.InOrStdin())

			configDir := c.configDir
			if configDir == "" {
				configDir = coopConfigFolder()
			}
			store, err := coop.NewStore(configDir)
			if err != nil {
				// Never fail an agent's turn over hook infrastructure.
				return outputStopHookDecision(cmd.OutOrStdout(), stopHookDecision{})
			}
			return runCoopStopHook(cmd.OutOrStdout(), store, workflow.NewService(store), c.session)
		},
	}
	c.cmd.Flags().StringVar(&c.session, "session", "", "Session ID (defaults to the only active session, if there is exactly one)")
	c.cmd.Flags().StringVar(&c.configDir, "config-dir", "", "Stripe config folder holding the session store")
	configureAgentCommand(c.cmd)
	return c
}

// stopHookResumer is workflow.Service's read-only lifecycle query. The hook
// reuses it rather than recomputing the state machine: Resume already knows
// about aborted and completed sessions, rejected work, steps ready for review,
// and the next pending task, and an empty next is its "nothing to do" signal.
type stopHookResumer interface {
	Resume(sessionID string) (coop.CommandResponse, error)
}

func runCoopStopHook(out io.Writer, store stopHookStore, resumer stopHookResumer, sessionID string) error {
	session := resolveStopHookSession(store, sessionID, time.Now())
	if session == nil {
		return outputStopHookDecision(out, stopHookDecision{})
	}

	resume, err := resumer.Resume(session.ID)
	if err != nil || !resume.OK || resume.Next == "" {
		// Nothing actionable: let the turn end and reset the bookkeeping so a
		// later review starts fresh.
		_ = store.RemoveStopHookState(session.ID)
		return outputStopHookDecision(out, stopHookDecision{})
	}

	state, err := store.ReadStopHookState(session.ID)
	if err != nil {
		state = coop.StopHookState{}
	}
	if state.ObservedCommand != resume.Next {
		// The lifecycle moved to a different command, so the loop is working.
		state = coop.StopHookState{ObservedCommand: resume.Next}
	}
	if state.Blocks >= maxConsecutiveStopBlocks {
		// Keep the exhausted budget on disk. Clearing it here would let the very
		// next Stop event start from zero and block three more times, so the
		// agent could never actually stop until the lifecycle moved on.
		_ = store.WriteStopHookState(session.ID, state)
		return outputStopHookDecision(out, stopHookDecision{
			SystemMessage: fmt.Sprintf(
				"Co-op is still waiting on you for session %s, and the agent has stopped. To hand it back, run this in the agent pane:\n\n%s",
				session.ID, coop.ResumeCommand(session.ID)),
		})
	}

	state.Blocks++
	if err := store.WriteStopHookState(session.ID, state); err != nil {
		// The counter is the only thing bounding this loop. If it cannot be
		// persisted, blocking would repeat forever with no release, so fail
		// open rather than trapping the agent.
		return outputStopHookDecision(out, stopHookDecision{})
	}

	return outputStopHookDecision(out, stopHookDecision{
		Decision: "block",
		Reason: fmt.Sprintf(
			"This Co-op turn is not over: %s\nDo not stop and do not ask a question here — the developer is watching the TUI, not your pane. Run this now:\n\n%s",
			resume.Message, resume.Next),
	})
}

// resolveStopHookSession falls back to the single active session because in
// discovery mode ("stripe coop start" with no blueprint) no session exists when
// the hook is installed, so the launcher cannot bake an ID into the command.
//
// With more than one active session the fallback is ambiguous — sessions do not
// record which directory they belong to — and guessing would let this agent's
// hook hand back a command targeting somebody else's session, mutating it. Fail
// open instead: no session means no block.
func resolveStopHookSession(store stopHookStore, sessionID string, now time.Time) *coop.Session {
	if sessionID != "" {
		if session, err := store.Read(sessionID); err == nil {
			return session
		}
		return nil
	}
	active, err := store.ActiveSessions()
	if err != nil || len(active) != 1 {
		return nil
	}
	// Sessions only leave "active" through an explicit stop or completion, so an
	// abandoned run stays a candidate forever. Only adopt one this agent could
	// plausibly have just created.
	if now.Sub(active[0].UpdatedAt) > staleSessionFallbackAge {
		return nil
	}
	return active[0]
}

func outputStopHookDecision(out io.Writer, decision stopHookDecision) error {
	data, err := json.Marshal(decision)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, string(data))
	return nil
}

// stopHookCommand is the command an agent harness runs for the Stop event.
//
// The string is shell-quoted, so it assumes the harness runs hooks through a
// shell — Claude Code documents them as shell commands. The config folder still
// travels as an explicit flag rather than an "XDG_CONFIG_HOME=... " prefix:
// quoting survives a plain `sh -c`, but a leading environment assignment is a
// shell-only construct that would turn into argv[0] under any harness that
// execs directly.
func stopHookCommand(stripeBin, sessionID string) string {
	cmd := fmt.Sprintf("%s coop agent stop-hook --config-dir %s",
		shellQuote(stripeBin), shellQuote(coopConfigFolder()))
	if sessionID != "" {
		cmd += " --session " + shellQuote(sessionID)
	}
	return cmd
}
