package resource

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

//
// Public types
//

// ResourceCmd represents resource commands. Resource commands can be either
// top-level commands or nested under namespace commands. Resource commands
// are containers for operation commands.
//
// Example of resources: `customers`, `payment_intents` (top-level, not
// namespaced), `early_fraud_warnings` (namespaced under `radar`).
type ResourceCmd struct { //nolint:revive
	Cmd           *cobra.Command
	Name          string
	OperationCmds map[string]*OperationCmd
}

//
// Public functions
//

// GetResourceCmdName returns the name for the resource commands. This differs
// from the resource name because we want to use the pluralized name in most
// cases.
func GetResourceCmdName(name string) string {
	switch name {
	case "balance":
		// `balance` is a singleton resource and is not pluralized
		return "balance"
	case "capability":
		return "capabilities"
	case "three_d_secure":
		return "3d_secure"
	case "usage_record_summary":
		return "usage_record_summaries"
	default:
		return name + "s"
	}
}

// NewResourceCmd returns a new ResourceCmd.
func NewResourceCmd(parentCmd *cobra.Command, resourceName string) *ResourceCmd {
	cmd := &cobra.Command{
		Use:         resourceName,
		Annotations: make(map[string]string),
	}
	cmd.SetUsageTemplate(resourceUsageTemplate())

	parentCmd.AddCommand(cmd)
	parentCmd.Annotations[resourceName] = "resource"

	return &ResourceCmd{
		Cmd:           cmd,
		Name:          resourceName,
		OperationCmds: make(map[string]*OperationCmd),
	}
}

//
// Private functions
//

func resourceUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} <operation> [parameters...]{{end}}{{if gt (len .Aliases) 0}}

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
		ansi.Bold("Available Operations:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global Flags:"),
		ansi.Bold("Additional help topics:"),
	)
}
