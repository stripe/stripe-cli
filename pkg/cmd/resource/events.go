package resource

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
)

// AddEventsSubCmds adds custom subcommands to the `events` command created
// automatically as a resource command.
func AddEventsSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	eventsCmd, ok := cmdutil.FindSubCmd(rootCmd, "events")
	if !ok {
		return errors.New("could not find events command")
	}

	NewEventsResendCmd(eventsCmd, cfg)
	return nil
}
