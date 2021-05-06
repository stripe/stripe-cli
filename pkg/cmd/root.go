//go:generate go run ../gen/gen_resources_cmds.go
//go:generate go run ../gen/gen_events_list.go

package cmd

import (
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
	"github.com/stripe/stripe-cli/pkg/stripe"
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
		// if the device name errors, don't fail running the command
		deviceName, _ := Config.Profile.GetDeviceName()
		stripe.GetTelemetryInstance().SetDeviceName(deviceName)
		stripe.GetTelemetryInstance().SetCommandContext(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.SetUsageTemplate(getUsageTemplate())
	rootCmd.SetVersionTemplate(version.Template)

	if err := rootCmd.Execute(); err != nil {
		errString := err.Error()
		isLoginRequiredError := errString == validators.ErrAPIKeyNotConfigured.Error() || errString == validators.ErrDeviceNameNotConfigured.Error()

		switch {
		case isLoginRequiredError:
			// capitalize first letter of error because linter
			errRunes := []rune(errString)
			errRunes[0] = unicode.ToUpper(errRunes[0])

			fmt.Printf("%s. Running `stripe login`...\n", string(errRunes))
			loginCommand, _, err := rootCmd.Find([]string{"login"})

			if err != nil {
				fmt.Println(err)
			}

			err = loginCommand.RunE(&cobra.Command{}, []string{})

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
