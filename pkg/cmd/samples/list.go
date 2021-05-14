package samples

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/samples"
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
		RunE: listCmd.runListCmd,
	}

	return listCmd
}

func (lc *ListCmd) runListCmd(cmd *cobra.Command, args []string) error {
	fmt.Println("A list of available Stripe Samples:")
	fmt.Println()

	spinner := ansi.StartNewSpinner("Loading...", os.Stdout)

	list, err := samples.GetSamples("list")
	if err != nil {
		ansi.StopSpinner(spinner, "Error: please check your internet connection and try again!", os.Stdout)
		return err
	}
	ansi.StopSpinner(spinner, "", os.Stdout)

	names := samples.Names(list)
	sort.Strings(names)

	for _, name := range names {
		fmt.Println(list[name].BoldName())
		fmt.Println(list[name].Description)
		fmt.Printf("Repo: %s\n", list[name].URL)
		fmt.Println()
	}

	return nil
}
