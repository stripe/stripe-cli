package coopcmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/coop"
)

type coopStatusCmd struct {
	cmd     *cobra.Command
	session string
	asJSON  bool
}

func newCoopStatusCmd() *coopStatusCmd {
	sc := &coopStatusCmd{}
	sc.cmd = &cobra.Command{
		Use:   "status",
		Short: "Show the current co-op session status",
		Long:  `Displays a summary of the current co-op session including step progress.`,
		RunE:  sc.runStatusCmd,
	}

	sc.cmd.Flags().StringVar(&sc.session, "session", "", "Session ID (defaults to latest)")
	sc.cmd.Flags().BoolVar(&sc.asJSON, "json", false, "Output in JSON format")

	return sc
}

func (sc *coopStatusCmd) runStatusCmd(cmd *cobra.Command, args []string) error {
	store, err := coop.NewStore(coopConfigFolder())
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	var session *coop.Session
	if sc.session != "" {
		session, err = store.Read(sc.session)
	} else {
		session, err = store.LatestSession()
	}
	if err != nil {
		return outputCoopError("No active session found. Start one with 'stripe coop start <blueprint>'.", "stripe coop start one-time-payment")
	}

	if sc.asJSON {
		return outputJSON(session)
	}

	summary := session.NodeSummary()
	total := session.TotalNodes()

	fmt.Printf("Session: %s\n", session.ID)
	fmt.Printf("Blueprint: %s\n", session.Blueprint)
	fmt.Printf("Status: %s\n", session.Status)
	fmt.Printf("Progress: %d/%d nodes complete\n", summary[coop.NodeDone], total)
	fmt.Println()

	if summary[coop.NodeActive] > 0 {
		fmt.Printf("  Active: %d\n", summary[coop.NodeActive])
	}
	if summary[coop.NodeReview] > 0 {
		fmt.Printf("  Awaiting review: %d\n", summary[coop.NodeReview])
	}
	if summary[coop.NodePending] > 0 {
		fmt.Printf("  Pending: %d\n", summary[coop.NodePending])
	}
	if summary[coop.NodeSkipped] > 0 {
		fmt.Printf("  Skipped: %d\n", summary[coop.NodeSkipped])
	}

	fmt.Println()
	fmt.Printf("Join the live view: stripe coop join %s\n", session.ID)

	return nil
}
