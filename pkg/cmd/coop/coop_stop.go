package coopcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopStopCmd struct {
	cmd     *cobra.Command
	session string
	abort   bool
}

func newCoopStopCmd() *coopStopCmd {
	sc := &coopStopCmd{}
	sc.cmd = &cobra.Command{
		Use:   "stop",
		Short: "End the current co-op session",
		Long: `Ends the current co-op session. By default marks it as "completed"
(integration finished successfully). Use --abort to mark it as "aborted"
(stopped early, integration incomplete).

Ended sessions won't be picked up by co-op join or agent lifecycle commands.`,
		RunE: sc.runStopCmd,
	}

	sc.cmd.Flags().StringVar(&sc.session, "session", "", "Session ID (defaults to latest active)")
	sc.cmd.Flags().BoolVar(&sc.abort, "abort", false, "Mark session as aborted (incomplete, not finishing) instead of completed (done successfully)")

	return sc
}

func (sc *coopStopCmd) runStopCmd(cmd *cobra.Command, args []string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var session *coop.Session
	if sc.session != "" {
		session, err = store.Read(sc.session)
	} else {
		session, err = store.LatestActiveSession()
	}
	if err != nil {
		return outputCoopError("No active session found.", "stripe coop start <blueprint>")
	}

	if sc.abort {
		session.Status = coop.SessionAborted
	} else {
		session.Status = coop.SessionCompleted
	}

	if err := store.Write(session); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	status := string(session.Status)
	return outputJSON(coop.CommandResponse{
		OK:        true,
		SessionID: session.ID,
		State:     status,
		Message:   fmt.Sprintf("Session %s: %s", status, session.ID),
	})
}
