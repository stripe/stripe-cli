package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/spec"
	"github.com/stripe/stripe-cli/pkg/version"
)

// Config is the cli configuration for the user
var Config config.Config

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
	Short:   "A CLI to help you integrate Stripe with your application",
	Long: fmt.Sprintf(`%s

The Stripe CLI gives you tools to help build with Stripe. You can do things like
connect to a Stripe webhook tunnel and test webhooks locally, make test mode
requests to the API, and trigger certain webhook events.

Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`,
		ansi.Italic("⚠️  The Stripe CLI is in beta! Have feedback? Let us know, run: 'stripe feedback'. ⚠️"),
	),
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetUsageTemplate(fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{if .Annotations}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "api")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (eq (index $.Annotations $cmd.Name) "webhooks")}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

%s{{range $index, $cmd := .Commands}}{{if (not (index $.Annotations $cmd.Name))}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}{{else}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

%s{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("API commands:"),
		ansi.Bold("Webhook commands:"),
		ansi.Bold("Other commands:"),
		ansi.Bold("Available commands:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global flags:"),
		ansi.Bold("Additional help topics:"),
	))
	rootCmd.SetVersionTemplate(version.Template)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(Config.InitConfig)

	rootCmd.PersistentFlags().String("api-key", "", "Your test mode API secret key to use for the command")
	rootCmd.PersistentFlags().StringVar(&Config.Color, "color", "auto", "turn on/off color output (on, off, auto)")
	rootCmd.PersistentFlags().StringVar(&Config.ProfilesFile, "config", "", "config file (default is $HOME/.config/stripe/config.toml)")
	rootCmd.PersistentFlags().StringVar(&Config.LogLevel, "log-level", "info", "log level (debug, info, warn, error)")

	rootCmd.PersistentFlags().StringVar(&Config.Profile.ProfileName, "project-name", "default", "the project name to read from for config")
	rootCmd.PersistentFlags().StringVar(&Config.Profile.DeviceName, "device-name", "", "device name")
	rootCmd.Flags().BoolP("version", "v", false, "Get the version of the Stripe CLI")

	viper.BindPFlag("secret_key", rootCmd.PersistentFlags().Lookup("api-key")) // #nosec G104

	viper.SetEnvPrefix("stripe")
	viper.AutomaticEnv() // read in environment variables that match

	rootCmd.AddCommand(newCompletionCmd().cmd)
	rootCmd.AddCommand(newLoginCmd().cmd)
	rootCmd.AddCommand(newDeleteCmd().reqs.Cmd)
	rootCmd.AddCommand(newFeedbackdCmd().cmd)
	rootCmd.AddCommand(newGetCmd().reqs.Cmd)
	rootCmd.AddCommand(newListenCmd().cmd)
	rootCmd.AddCommand(newPostCmd().reqs.Cmd)
	rootCmd.AddCommand(newStatusCmd().cmd)
	rootCmd.AddCommand(newTriggerCmd().cmd)
	rootCmd.AddCommand(newVersionCmd().cmd)

	stripeAPI, err := spec.LoadSpec("")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse OpenAPI spec")
		os.Exit(1)
	}
	addAllResourceCmds(rootCmd, stripeAPI)
}
