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

var runTmuxOutput = func(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).CombinedOutput()
	return string(out), err
}

type tmuxAgentResumer struct {
	tuiPane  string
	keyDelay time.Duration
	mu       sync.Mutex
}

func newTmuxAgentResumer(tuiPane string) *tmuxAgentResumer {
	return &tmuxAgentResumer{tuiPane: tuiPane, keyDelay: 100 * time.Millisecond}
}

func (r *tmuxAgentResumer) Notify(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	pane, err := r.findAgentPane()
	if err != nil {
		return err
	}
	resumeCommand := coop.ResumeCommand(sessionID)
	prompt := fmt.Sprintf("Human input updated Stripe Co-op session %s. Resume neutrally: run exactly `%s`. If its JSON has a non-empty `next`, run that command exactly; otherwise continue your current work.", sessionID, resumeCommand)
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
	out, err := runTmuxOutput("list-panes", "-t", r.tuiPane, "-F", "#{pane_id}\t#{@stripe_coop_agent}")
	if err != nil {
		return "", fmt.Errorf("listing Co-op tmux panes: %w", err)
	}
	var panes []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		pane, tag, ok := strings.Cut(line, "\t")
		if ok && strings.TrimSpace(tag) == "1" {
			panes = append(panes, strings.TrimSpace(pane))
		}
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
