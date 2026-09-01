package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// isHelpRequest reports whether the raw, unparsed args passed to a
// DisableFlagParsing command are a request for help rather than a real
// invocation.
func isHelpRequest(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

// provisionPluginName is the plugin that backs `stripe provision`.
const provisionPluginName = "projects"

// provisionPluginSubcommand is the projects plugin subcommand that
// `stripe provision` aliases to.
const provisionPluginSubcommand = "add"

// buildProvisionPluginArgs prepends the aliased plugin subcommand to the raw
// args forwarded from `stripe provision`.
func buildProvisionPluginArgs(args []string) []string {
	return append([]string{provisionPluginSubcommand}, args...)
}

type provisionCmd struct {
	cmd *cobra.Command

	runProvisionCmdFn func(cmd *cobra.Command, args []string) error
}

// newProvisionCmd is a thin alias for `stripe projects add`: it ensures the
// projects plugin is installed, then forwards all args/flags to it.
func newProvisionCmd() *provisionCmd {
	pc := &provisionCmd{}
	pc.runProvisionCmdFn = pc.runProvisionCmd

	pc.cmd = &cobra.Command{
		Use:   "provision [service]",
		Short: "Provision a service (alias for `stripe projects add`)",
		Long: `Provision a service via the projects plugin.

This is an alias for 'stripe projects add' — all flags are forwarded as-is.
If the projects plugin isn't installed yet, it will be installed
automatically before running.`,
		Example: `stripe provision <provider>/<service>
  stripe provision neon/postgres --config '{"region":"iad1"}'
  stripe provision agentmail/api --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.runProvisionCmdFn(cmd, args)
		},
		DisableFlagParsing: true,
	}

	// Currently `stripe provision` only supports some of the flags supported
	// by `stripe projects add`; actual parsing/forwarding happens
	// raw in runProvisionCmd since DisableFlagParsing is set above.
	flags := pc.cmd.Flags()
	flags.String("name", "", "Logical resource name used for local resource references and environment variable prefixes")
	flags.String("config", "", "Service configuration as a JSON string")
	flags.Bool("json", false, "Output structured JSON and suppress interactive prompts")
	flags.BoolP("yes", "y", false, "Skip confirmation prompts")
	flags.Bool("accept-tos", false, "Accept provider terms of service without prompting")

	return pc
}

func (pc *provisionCmd) runProvisionCmd(cmd *cobra.Command, args []string) error {
	if isHelpRequest(args) {
		return cmd.Help()
	}

	ctx := withSIGTERMCancel(commandContextOrBackground(cmd), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.provisionCmd.runProvisionCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	fs := afero.NewOsFs()

	plugin, err := plugins.LookUpPlugin(ctx, &Config, fs, provisionPluginName)
	if err != nil {
		return err
	}

	pluginArgs := buildProvisionPluginArgs(args)

	err = plugin.Run(ctx, &Config, fs, pluginArgs, "", "", "", "", "")
	plugins.CleanupAllClients()
	if err == nil {
		dashboardBaseURL := stripe.DashboardBaseURLForAPIBaseURL(stripe.DefaultAPIBaseURL)
		plugins.CheckLatestPluginVersion(ctx, &Config, fs, plugin, stripe.DefaultAPIBaseURL, dashboardBaseURL)
	}

	if err != nil {
		if err == validators.ErrAPIKeyNotConfigured {
			return errorcategory.New(errorcategory.Auth, "provision failed due to API key not configured, please run `stripe login` or specify the `--api-key`")
		}

		log.WithFields(log.Fields{
			"prefix": "provisionCmd.runProvisionCmd",
		}).Debug(fmt.Sprintf("Provision command exited with error: %s", err))

		os.Exit(1)
	}

	return nil
}
