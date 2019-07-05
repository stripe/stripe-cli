package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	prof "github.com/stripe/stripe-cli/profile"
	"github.com/stripe/stripe-cli/version"
)

// Profile is the cli configuration for the user
var Profile prof.Profile

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "stripe",
	SilenceUsage:  true,
	SilenceErrors: true,
	Annotations: map[string]string{
		"get":     "api",
		"post":    "api",
		"delete":  "api",
		"trigger": "webhooks",
		"listen":  "webhooks",
	},
	Version: version.Version,
	Short:   "A CLI to help you develop your application with Stripe",
	Long: `The Stripe CLI gives you tools to make integrating your application
with Stripe easier. You do things like connect to a Stripe webhook tunnel to
test webhooks locally, make test mode requests to the API, and trigger certain
webhook events.

Before you use the CLI, you'll need to configure it:
$ stripe configure

If you're working on multiple projects, you can run the configure command with the
--project-name flag:
$ stripe configure --project-name rocket-rides`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetUsageTemplate(`Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if .Annotations}}

API commands:{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "api")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

Webhook commands:{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "webhooks")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

Other commands:{{range $index, $cmd := .Commands}}{{if (not (index $.Annotations $cmd.Name))}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}{{else}}

Available commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(Profile.InitConfig)

	rootCmd.PersistentFlags().String("api-key", "", "Your test mode API secret key to use for the command")
	rootCmd.PersistentFlags().StringVar(&Profile.Color, "color", "auto", "turn on/off color output (on, off, auto)")
	rootCmd.PersistentFlags().StringVar(&Profile.ConfigFile, "config", "", "config file (default is $HOME/.config/stripe/config.toml)")
	rootCmd.PersistentFlags().StringVar(&Profile.ProfileName, "project-name", "default", "the project name to read from for config")
	rootCmd.PersistentFlags().StringVar(&Profile.LogLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&Profile.DeviceName, "device-name", "", "device name")
	viper.BindPFlag("secret_key", rootCmd.PersistentFlags().Lookup("api-key")) // #nosec G104

	viper.SetEnvPrefix("stripe")
	viper.AutomaticEnv() // read in environment variables that match

	rootCmd.AddCommand(newCompletionCmd().cmd)
	rootCmd.AddCommand(newLoginCmd().cmd)
	rootCmd.AddCommand(newDeleteCmd().reqs.Cmd)
	rootCmd.AddCommand(newGetCmd().reqs.Cmd)
	rootCmd.AddCommand(newListenCmd().cmd)
	rootCmd.AddCommand(newPostCmd().reqs.Cmd)
	rootCmd.AddCommand(newTriggerCmd().cmd)
	rootCmd.AddCommand(newVersionCmd().cmd)
}
