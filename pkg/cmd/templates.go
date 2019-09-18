package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
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

func getLogin(fs *afero.Fs, cfg *config.Config) string {
	// We're checking against the path because we don't initialize the config
	// at this point of execution.
	path := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	file := filepath.Join(path, "config.toml")

	exists, _ := afero.Exists(*fs, file)

	if !exists {
		return `
Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`
	}

	return ""
}

func getUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if .Annotations}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "webhooks")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "stripe")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "resources")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (not (index $.Annotations $cmd.Name))}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}{{else}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{WrappedLocalFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{WrappedInheritedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("Webhook commands:"),
		ansi.Bold("Stripe commands:"),
		ansi.Bold("Resource commands:"),
		ansi.Bold("Other commands:"),
		ansi.Bold("Available commands:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global flags:"),
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
