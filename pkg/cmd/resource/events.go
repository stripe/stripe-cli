package resource

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

// AddEventsSubCmds adds custom subcommands to the `events` command created
// automatically as a resource command.
func AddEventsSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "events" {
			found = true

			NewEventsResendCmd(cmd, cfg)

			break
		}
	}

	if !found {
		return errors.New("Could not find events command")
	}

	return nil
}
