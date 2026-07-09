package coopcmd

import (
	"fmt"
	"sort"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
	"github.com/stripe/stripe-cli/pkg/coop/tui"
)

type coopJoinCmd struct {
	cmd    *cobra.Command
	resume bool
	wait   bool
}

func newCoopJoinCmd() *coopJoinCmd {
	jc := &coopJoinCmd{}
	jc.cmd = &cobra.Command{
		Use:   "join [session-id]",
		Short: "Join a co-op session and watch progress in the terminal UI",
		Long: `Launches the co-op terminal UI to watch an agent work through a blueprint
in real time. Press 'c' to confirm completed steps, 'e' to expand details.

If no session ID is given, joins the most recently updated active session.
Use --resume to pick from all recent sessions.`,
		Args: cobra.MaximumNArgs(1),
		RunE: jc.runJoinCmd,
	}

	jc.cmd.Flags().BoolVar(&jc.resume, "resume", false, "Pick from recent sessions")
	jc.cmd.Flags().BoolVar(&jc.wait, "wait", false, "Wait for a new session to be created (used by coop run)")
	jc.cmd.Flags().MarkHidden("wait")

	return jc
}

func (jc *coopJoinCmd) runJoinCmd(cmd *cobra.Command, args []string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	// Wait mode: launch TUI immediately and poll for a new session
	if jc.wait {
		existingIDs := make(map[string]bool)
		if ids, err := store.List(); err == nil {
			for _, id := range ids {
				existingIDs[id] = true
			}
		}
		return tui.RunWaiting(store, existingIDs, tui.WithSandboxClaimURL(coopSandboxClaimURL()))
	}

	var session *coop.Session

	switch {
	case len(args) > 0:
		session, err = store.Read(args[0])
		if err != nil {
			return fmt.Errorf("session %q not found: %w", args[0], err)
		}
	case jc.resume:
		session, err = jc.pickSession(store)
		if err != nil {
			return err
		}
	default:
		// Try active first, then fall back to latest (including completed sessions with next-action suggestions).
		session, err = store.LatestActiveSession()
		if err != nil {
			session, err = store.LatestSession()
		}
		if err != nil {
			return fmt.Errorf("no session found. Use --resume to pick from recent sessions, or start a new one with 'stripe coop start'")
		}
	}

	// Show reconnection notice if session is already in progress
	summary := session.NodeSummary()
	if summary[coop.NodeDone] > 0 || summary[coop.NodeActive] > 0 || summary[coop.NodeReview] > 0 {
		fmt.Printf("Reconnecting to %s (%s) — %d/%d done\n",
			session.ID, session.Blueprint, summary[coop.NodeDone], session.TotalNodes())
		fmt.Println()
	}

	return tui.Run(store, session.ID, tui.WithSandboxClaimURL(coopSandboxClaimURL()))
}

type sessionChoice struct {
	session *coop.Session
	label   string
}

func (jc *coopJoinCmd) pickSession(store *coop.Store) (*coop.Session, error) {
	entries, err := recentSessionChoices(store)
	if err != nil {
		return nil, err
	}

	var options []huh.Option[string]
	for _, e := range entries {
		options = append(options, huh.NewOption(e.label, e.session.ID))
	}

	var choice string
	err = selectString("Pick a session to resume:", options, &choice)
	if err != nil {
		return nil, err
	}

	return store.Read(choice)
}

func recentSessionChoices(store *coop.Store) ([]sessionChoice, error) {
	ids, err := store.List()
	if err != nil || len(ids) == 0 {
		return nil, fmt.Errorf("no sessions found. Start one with 'stripe coop start <blueprint>'")
	}

	var entries []sessionChoice

	for _, id := range ids {
		s, err := store.Read(id)
		if err != nil {
			continue
		}
		summary := s.NodeSummary()
		status := string(s.Status)
		label := fmt.Sprintf("%s  %s  %d/%d done  [%s]",
			s.ID, s.Blueprint, summary[coop.NodeDone], s.TotalNodes(), status)
		entries = append(entries, sessionChoice{session: s, label: label})
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no readable sessions found")
	}

	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].session.UpdatedAt.Equal(entries[j].session.UpdatedAt) {
			return entries[i].session.ID < entries[j].session.ID
		}
		return entries[i].session.UpdatedAt.After(entries[j].session.UpdatedAt)
	})

	return entries, nil
}
