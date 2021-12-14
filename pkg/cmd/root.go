//go:generate go run ../gen/gen_resources_cmds.go
//go:generate go run ../gen/gen_events_list.go

package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

// Config is the cli configuration for the user
var Config config.Config

var fs = afero.NewOsFs()

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:           "stripe",
	SilenceUsage:  true,
	SilenceErrors: true,
	Annotations: map[string]string{
		"get":       "http",
		"post":      "http",
		"delete":    "http",
		"trigger":   "webhooks",
		"listen":    "webhooks",
		"logs":      "stripe",
		"status":    "stripe",
		"resources": "resources",
	},
	Version: version.Version,
	Short:   "A CLI to help you integrate Stripe with your application",
	Long: fmt.Sprintf(`The official command-line tool to interact with Stripe.
%s`,
		getLogin(&fs, &Config),
	),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// if getting the config errors, don't fail running the command
		merchant, _ := Config.Profile.GetAccountID()
		telemetryMetadata := stripe.GetEventMetadata(cmd.Context())
		telemetryMetadata.SetCobraCommandContext(cmd)
		telemetryMetadata.SetMerchant(merchant)
		telemetryMetadata.SetUserAgent(useragent.GetEncodedUserAgent())

		// record command invocation
		sendCommandInvocationEvent(cmd.Context())
	},
}

func sendCommandInvocationEvent(ctx context.Context) {
	telemetryClient := stripe.GetTelemetryClient(ctx)
	if telemetryClient != nil {
		go telemetryClient.SendEvent(ctx, "Command Invoked", "Cobra")
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute(ctx context.Context) {
	telemetryMetadata := stripe.NewEventMetadata()
	updatedCtx := stripe.WithEventMetadata(ctx, telemetryMetadata)

	rootCmd.SetUsageTemplate(getUsageTemplate())
	rootCmd.SetVersionTemplate(version.Template)
	if err := rootCmd.ExecuteContext(updatedCtx); err != nil {
		errString := err.Error()
		isLoginRequiredError := errString == validators.ErrAPIKeyNotConfigured.Error() || errString == validators.ErrDeviceNameNotConfigured.Error()

		switch {
		case requests.IsAPIKeyExpiredError(err):
			fmt.Fprintln(os.Stderr, "The API key provided has expired. Obtain a new key from the Dashboard or run `stripe login` and try again.")
		case isLoginRequiredError:
			// capitalize first letter of error because linter
			errRunes := []rune(errString)
			errRunes[0] = unicode.ToUpper(errRunes[0])

			fmt.Printf("%s. Running `stripe login`...\n", string(errRunes))

			err = login.Login(updatedCtx, stripe.DefaultDashboardBaseURL, &Config, os.Stdin)

			if err != nil {
				fmt.Println(err)
			}

		case strings.Contains(errString, "unknown command"):
			suggStr := "\nS"

			suggestions := rootCmd.SuggestionsFor(os.Args[1])
			if len(suggestions) > 0 {
				suggStr = fmt.Sprintf(" Did you mean \"%s\"?\nIf not, s", suggestions[0])
			}

			fmt.Println(fmt.Sprintf("Unknown command \"%s\" for \"%s\".%s"+
				"ee \"stripe --help\" for a list of available commands.",
				os.Args[1], rootCmd.CommandPath(), suggStr))

		default:
			fmt.Println(err)
		}

		os.Exit(1)
	} else {
		userInput := os.Args[1:]
		// --color on/off/auto
		if len(userInput) == 2 && userInput[0] == "--color" {
			fmt.Println("You provided the \"--color\" flag but did not specify any command. The \"--color\" flag configures the color output of a specified command.")
		}
	}
}

func init() {
	cobra.OnInitialize(Config.InitConfig)

	rootCmd.PersistentFlags().StringVar(&Config.Profile.APIKey, "api-key", "", "Your API key to use for the command")
	rootCmd.PersistentFlags().StringVar(&Config.Color, "color", "", "turn on/off color output (on, off, auto)")
	rootCmd.PersistentFlags().StringVar(&Config.ProfilesFile, "config", "", "config file (default is $HOME/.config/stripe/config.toml)")
	rootCmd.PersistentFlags().StringVar(&Config.Profile.DeviceName, "device-name", "", "device name")
	rootCmd.PersistentFlags().StringVar(&Config.LogLevel, "log-level", "info", "log level (debug, info, trace, warn, error)")
	rootCmd.PersistentFlags().StringVarP(&Config.Profile.ProfileName, "project-name", "p", "default", "the project name to read from for config")
	rootCmd.Flags().BoolP("version", "v", false, "Get the version of the Stripe CLI")

	viper.BindPFlag("color", rootCmd.PersistentFlags().Lookup("color"))

	rootCmd.AddCommand(newCompletionCmd().cmd)
	rootCmd.AddCommand(newConfigCmd().cmd)
	rootCmd.AddCommand(newDaemonCmd(&Config).cmd)
	rootCmd.AddCommand(newDeleteCmd().reqs.Cmd)
	rootCmd.AddCommand(newFeedbackdCmd().cmd)
	rootCmd.AddCommand(newFixturesCmd(&Config).Cmd)
	rootCmd.AddCommand(newGetCmd().reqs.Cmd)
	rootCmd.AddCommand(newListenCmd().cmd)
	rootCmd.AddCommand(newLoginCmd().cmd)
	rootCmd.AddCommand(newLogoutCmd().cmd)
	rootCmd.AddCommand(newLogsCmd(&Config).Cmd)
	rootCmd.AddCommand(newOpenCmd().cmd)
	rootCmd.AddCommand(newPostCmd().reqs.Cmd)
	rootCmd.AddCommand(newResourcesCmd().cmd)
	rootCmd.AddCommand(newSamplesCmd().cmd)
	rootCmd.AddCommand(newServeCmd().cmd)
	rootCmd.AddCommand(newStatusCmd().cmd)
	rootCmd.AddCommand(newTriggerCmd().cmd)
	rootCmd.AddCommand(newVersionCmd().cmd)
	rootCmd.AddCommand(newPlaybackCmd().cmd)
	rootCmd.AddCommand(newPostinstallCmd(&Config).cmd)
	rootCmd.AddCommand(newCommunityCmd().cmd)

	addAllResourcesCmds(rootCmd)

	err := resource.AddEventsSubCmds(rootCmd, &Config)
	if err != nil {
		log.Fatal(err)
	}

	err = resource.AddTerminalSubCmds(rootCmd, &Config)
	if err != nil {
		log.Fatal(err)
	}
}
