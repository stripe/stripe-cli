package samples

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type listCmd struct {
	Cmd *cobra.Command
}

// NewListCmd defines a new samples command that lists all samples
func NewListCmd() *cobra.Command {
	listCmd := &listCmd{}
	listCmd.Cmd = &cobra.Command{
		Use:   "list",
		Args:  validators.NoArgs,
		Short: "list available Stripe samples",
		Long:  `A list of available Stripe Sample integrations`,
		Run:   listCmd.runListCmd,
	}

	return listCmd.Cmd
}

func (lc *listCmd) runListCmd(cmd *cobra.Command, args []string) {
	fmt.Println("A list of available Stripe Sample integrations:")
	fmt.Println()

	for _, sample := range samples.List {
		fmt.Println(sample.BoldName())
		fmt.Println(sample.Description)
		fmt.Println(fmt.Sprintf("Repo: %s", sample.URL))
		fmt.Println()
	}
}
