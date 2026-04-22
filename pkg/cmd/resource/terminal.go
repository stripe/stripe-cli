package resource

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/cmdutil"
	"github.com/stripe/stripe-cli/pkg/config"
)

// AddTerminalSubCmds adds custom subcommands to the `terminal` command created
// automatically as a resource command.
func AddTerminalSubCmds(rootCmd *cobra.Command, cfg *config.Config) error {
	terminalCmd, ok := cmdutil.FindSubCmd(rootCmd, "terminal")
	if !ok {
		return errors.New("could not find terminal command")
	}

	NewQuickstartCmd(terminalCmd, cfg)
	return nil
}
