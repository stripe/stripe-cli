package resource

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

// AddCheckoutSubCmds adds custom subcommands to the `checkout` command created
// automatically as a resource command.
func AddCheckoutSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "checkout" {
			found = true

			NewCheckoutRunCmd(cmd, cfg)

			break
		}
	}

	if !found {
		return errors.New("Could not find checkout command")
	}

	return nil
}
