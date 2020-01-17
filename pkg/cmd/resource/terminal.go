package resource

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
)

// AddTerminalSubCmds adds custom subcommands to the `terminal` command created
// automatically as a resource command.
func AddTerminalSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	found := false

	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "terminal" {
			found = true

			NewQuickstartCmd(cmd, cfg)

			break
		}
	}

	if !found {
		return errors.New("Could not find terminal command")
	}

	return nil
}
