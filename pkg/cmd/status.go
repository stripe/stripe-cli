package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/status"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type statusCmd struct {
	cmd *cobra.Command

	format      string
	hideSpinner bool
	poll        bool
	pollRate    int
	verbose     bool
}

func newStatusCmd() *statusCmd {
	sc := &statusCmd{}
	sc.cmd = &cobra.Command{
		Use:   "status",
		Args:  validators.NoArgs,
		Short: "Check the status of the Stripe API",
		Example: `stripe status
  stripe status --poll
  stripe status --poll --verbose`,
		RunE: sc.runStatusCmd,
	}

	sc.cmd.Flags().StringVar(&sc.format, "format", "default", "The format to print the status as (either 'default' or 'json')")
	sc.cmd.Flags().BoolVar(&sc.verbose, "verbose", false, "Show status for all Stripe systems")
	sc.cmd.Flags().BoolVar(&sc.poll, "poll", false, "Keep polling for status updates")
	sc.cmd.Flags().IntVar(&sc.pollRate, "poll-rate", 60, "How many seconds to wait between status updates (minimum: 5)")
	sc.cmd.Flags().BoolVar(&sc.hideSpinner, "hide-spinner", false, "Hide the loading spinner when polling")

	return sc
}

func (sc *statusCmd) runStatusCmd(cmd *cobra.Command, args []string) error {
	if sc.format != "json" {
		version.CheckLatestVersion()
	}

	if sc.pollRate < 5 {
		return fmt.Errorf("poll-rate must be at least 5 seconds, received %d", sc.pollRate)
	}

	if sc.format != "default" && sc.format != "json" {
		return fmt.Errorf("invalid format, must be one of 'default' or 'json', received %s", sc.format)
	}

	for {
		stripeStatus, err := status.GetStatus()
		if err != nil {
			return err
		}

		formattedStatus, err := stripeStatus.FormattedMessage(sc.format, sc.verbose)
		if err != nil {
			return err
		}

		fmt.Println(formattedStatus)

		if !sc.poll {
			break
		}

		if sc.hideSpinner {
			time.Sleep(time.Duration(sc.pollRate) * time.Second)
		} else {
			spinner := ansi.StartNewSpinner("", os.Stderr)
			time.Sleep(time.Duration(sc.pollRate) * time.Second)
			ansi.StopSpinner(spinner, "", os.Stderr)
		}
	}

	return nil
}
