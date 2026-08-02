package coopcmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/tui"
)

const coopAgentPaneOption = "@stripe_coop_agent"

// parkedHeartbeatFreshFor classifies the agent as parked in await-review. While
// parked it refreshes the session heartbeat every 500ms and reads the human's
// decision from the session file itself; only an agent that has timed out back
// to its idle prompt needs a keystroke wake-up.
const parkedHeartbeatFreshFor = 2 * time.Second

var runTmuxOutput = func(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return string(out), err
}

type tmuxAgentResumer struct {
	tuiPane string
	// heartbeatAge reports how long ago the agent refreshed the session
	// heartbeat; nil disables the parked-agent check.
	heartbeatAge func(sessionID string) (time.Duration, error)
	keyDelay     time.Duration
	mu           sync.Mutex
}

func newTmuxAgentResumer(tuiPane string) *tmuxAgentResumer {
	return &tmuxAgentResumer{tuiPane: tuiPane, keyDelay: 100 * time.Millisecond}
}

func (r *tmuxAgentResumer) Notify(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// A parked agent picks the decision up from the session file on its own.
	// Injecting keystrokes is only for an agent that has timed out — doing it
	// while the agent is live would append our prompt to whatever is in its
	// input box (including a half-typed human message) or blindly submit an
	// open dialog.
	if r.heartbeatAge != nil {
		if age, err := r.heartbeatAge(sessionID); err == nil && age <= parkedHeartbeatFreshFor {
			return nil
		}
	}

	pane, err := r.findAgentPane()
	if err != nil {
		return err
	}
	resumeCommand := coop.ResumeCommand(sessionID)
	prompt := fmt.Sprintf("Human input updated Stripe Co-op session %s. Resume neutrally: run exactly `%s`. If its JSON has a non-empty `next`, run that command exactly; otherwise continue your current work.", sessionID, resumeCommand)
	// Clear any half-typed input first. C-u kills the line in readline-style
	// inputs and is a no-op in dialogs — unlike Escape, which can dismiss a
	// dialog the user needed to answer.
	if err := runTmux("send-keys", "-t", pane, "C-u"); err != nil {
		return fmt.Errorf("clearing Co-op agent input: %w", err)
	}
	if err := runTmux("send-keys", "-t", pane, "-l", prompt); err != nil {
		return fmt.Errorf("sending Co-op resume prompt: %w", err)
	}
	// Agent TUIs can coalesce rapid input as a paste. Give the literal text one
	// bounded beat to land before submitting it as a turn.
	if r.keyDelay > 0 {
		time.Sleep(r.keyDelay)
	}
	if err := runTmux("send-keys", "-t", pane, "Enter"); err != nil {
		return fmt.Errorf("submitting Co-op resume prompt: %w", err)
	}
	return nil
}

func (r *tmuxAgentResumer) findAgentPane() (string, error) {
	out, err := runTmuxOutput("list-panes", "-t", r.tuiPane, "-F", "#{pane_id}\t#{@stripe_coop_agent}\t#{pane_dead}")
	if err != nil {
		return "", fmt.Errorf("listing Co-op tmux panes: %w", err)
	}
	var panes []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 || strings.TrimSpace(fields[1]) != "1" {
			continue
		}
		// send-keys into a dead pane (remain-on-exit) succeeds silently and
		// the decision never reaches an agent. Fail so the TUI surfaces the
		// manual resume command instead.
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) == "1" {
			return "", fmt.Errorf("the Co-op agent pane has exited")
		}
		panes = append(panes, strings.TrimSpace(fields[0]))
	}
	if len(panes) != 1 {
		return "", fmt.Errorf("expected one Co-op agent pane, found %d", len(panes))
	}
	return panes[0], nil
}

func coopTUIOptions() []tui.Option {
	options := []tui.Option{tui.WithSandboxClaimURL(coopSandboxClaimURL())}
	if tuiPane := os.Getenv("TMUX_PANE"); tuiPane != "" {
		resumer := newTmuxAgentResumer(tuiPane)
		if store, err := coop.NewStore(coopConfigFolder()); err == nil {
			resumer.heartbeatAge = store.HeartbeatAge
		}
		options = append(options, tui.WithReviewDecisionNotifier(resumer.Notify))
	}
	return options
}

func splitCoopAgentPane(args ...string) (string, error) {
	args = append([]string{"split-window", "-P", "-F", "#{pane_id}"}, args...)
	out, err := runTmuxOutput(args...)
	if err != nil {
		return "", err
	}
	pane := strings.TrimSpace(out)
	if pane == "" {
		return "", fmt.Errorf("tmux split-window returned an empty pane ID")
	}
	if err := runTmux("set-option", "-p", "-t", pane, coopAgentPaneOption, "1"); err != nil {
		_ = runTmux("kill-pane", "-t", pane)
		return "", fmt.Errorf("tagging Co-op agent pane: %w", err)
	}
	return pane, nil
}
