//go:build wasm
// +build wasm

package samples

import (
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/validators"
)

// ListCmd prints a list of all the available sample projects that users can
// generate
type ListCmd struct {
	Cmd *cobra.Command
}

// NewListCmd creates and returns a list command for samples
func NewListCmd() *ListCmd {
	listCmd := &ListCmd{}
	listCmd.Cmd = &cobra.Command{
		Use:   "list",
		Args:  validators.NoArgs,
		Short: "List Stripe Samples supported by the CLI",
		Long: `A list of available Stripe Sample integrations that can be setup and bootstrap by
the CLI.`,
	}

	return listCmd
}
