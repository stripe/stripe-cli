package resource

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

//
// Public types
//

// NamespaceCmd represents namespace commands. Namespace commands are top-level
// commands that are simply containers for resource commands.
//
// Example of namespaces: `issuing`, `radar`, `terminal`.
type NamespaceCmd struct {
	Cmd          *cobra.Command
	Name         string
	ResourceCmds map[string]*ResourceCmd
}

//
// Public functions
//

// NewNamespaceCmd returns a new NamespaceCmd.
func NewNamespaceCmd(rootCmd *cobra.Command, namespaceName string) *NamespaceCmd {
	cmd := &cobra.Command{
		Use:         namespaceName,
		Annotations: make(map[string]string),
	}
	cmd.SetUsageTemplate(namespaceUsageTemplate())

	// For non-namespaced resources, we create a namespace command with the
	// empty string as its name so we can group the resource commands in its
	// ResourceCmds map, but we don't actually add the Cobra command as a
	// subcommand.
	if namespaceName != "" {
		rootCmd.AddCommand(cmd)
		rootCmd.Annotations[namespaceName] = "namespace"
	}

	return &NamespaceCmd{
		Cmd:          cmd,
		Name:         namespaceName,
		ResourceCmds: make(map[string]*ResourceCmd),
	}
}

//
// Private functions
//

func namespaceUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <resource> <operation> [parameters...]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{WrappedLocalFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{WrappedInheritedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("Available Resources:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}
