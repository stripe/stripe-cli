package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type resourcesCmd struct {
	cmd *cobra.Command
}

func newResourcesCmd() *resourcesCmd {
	rc := &resourcesCmd{}

	rc.cmd = &cobra.Command{
		Use:   "resources",
		Args:  validators.NoArgs,
		Short: "List namespace and resource subcommands",
	}
	rc.cmd.SetHelpTemplate(getResourcesHelpTemplate())

	return rc
}

func getResourcesHelpTemplate() string {
	// This template uses `.Parent` to access subcommands on the root command.
	return fmt.Sprintf(`%s

%s{{range $index, $cmd := .Parent.Commands}}{{if (eq (index $.Parent.Annotations $cmd.Name) "namespace")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Parent.Commands}}{{if (eq (index $.Parent.Annotations $cmd.Name) "resource")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

Use "stripe [command] --help" for more information about a command.
`,
		getBanner(),
		ansi.Bold("Available Namespaces:"),
		ansi.Bold("Available Resources:"),
	)
}
