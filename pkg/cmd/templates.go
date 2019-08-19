package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

//
// Public functions
//

// WrappedInheritedFlagUsages returns a string containing the usage information
// for all flags which were inherited from parent commands, wrapped to the
// terminal's width.
func WrappedInheritedFlagUsages(cmd *cobra.Command) string {
	return cmd.InheritedFlags().FlagUsagesWrapped(getTerminalWidth())
}

// WrappedLocalFlagUsages returns a string containing the usage information
// for all flags specifically set in the current command, wrapped to the
// terminal's width.
func WrappedLocalFlagUsages(cmd *cobra.Command) string {
	return cmd.LocalFlags().FlagUsagesWrapped(getTerminalWidth())
}

//
// Private functions
//

func getBanner() string {
	return ansi.Italic("⚠️  The Stripe CLI is in beta! Share your feedback with `stripe feedback` ⚠️")
}

func getUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if .Annotations}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "http")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "webhooks")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "stripe")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (not (index $.Annotations $cmd.Name))}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}{{else}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

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
		ansi.Bold("HTTP commands:"),
		ansi.Bold("Webhook commands:"),
		ansi.Bold("Stripe commands:"),
		ansi.Bold("Other commands:"),
		ansi.Bold("Available commands:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global flags:"),
		ansi.Bold("Additional help topics:"),
	)
}

func getTerminalWidth() int {
	var width int
	width, _, err := terminal.GetSize(0)
	if err != nil {
		width = 80
	}
	return width
}

func init() {
	cobra.AddTemplateFunc("WrappedInheritedFlagUsages", WrappedInheritedFlagUsages)
	cobra.AddTemplateFunc("WrappedLocalFlagUsages", WrappedLocalFlagUsages)
}
