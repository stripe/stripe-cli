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

	summary := session.StepSummary()
	total := session.TotalSteps()

	fmt.Printf("Session: %s\n", session.ID)
	fmt.Printf("Blueprint: %s\n", session.Blueprint)
	fmt.Printf("Status: %s\n", session.Status)
	fmt.Printf("Progress: %d/%d steps complete\n", summary[coop.StepDone], total)
	fmt.Println()

	if summary[coop.StepActive] > 0 {
		fmt.Printf("  Active: %d\n", summary[coop.StepActive])
	}
	if summary[coop.StepReview] > 0 {
		fmt.Printf("  Awaiting review: %d\n", summary[coop.StepReview])
	}
	if summary[coop.StepPending] > 0 {
		fmt.Printf("  Pending: %d\n", summary[coop.StepPending])
	}
	if summary[coop.StepSkipped] > 0 {
		fmt.Printf("  Skipped: %d\n", summary[coop.StepSkipped])
	}

	fmt.Println()
	fmt.Printf("Join the live view: stripe coop join %s\n", session.ID)

	return nil
}
