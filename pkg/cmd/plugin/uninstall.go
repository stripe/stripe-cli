package plugin

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// UninstallCmd is the struct used for configuring the plugin uninstall command
type UninstallCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
	fs  afero.Fs
	all bool
}

// hookBaseURLs holds the base URLs forwarded to a plugin's PreUninstall hook. Each
// field is empty unless the user explicitly passed the matching flag, so a plugin
// only hears about an override the user actually chose. The values are resolved once
// per invocation and shared by every plugin --all uninstalls.
type hookBaseURLs struct {
	api       string
	dashboard string
	access    string
}

// NewUninstallCmd creates a new command for uninstalling plugins
func NewUninstallCmd(config *config.Config) *UninstallCmd {
	uc := &UninstallCmd{}
	uc.fs = afero.NewOsFs()
	uc.cfg = config

	uc.Cmd = &cobra.Command{
		Use:   "uninstall [name]",
		Args:  uc.validateArgs,
		Short: "Uninstall a Stripe CLI plugin",
		Long: `Uninstall a Stripe CLI plugin.

Pass --all to uninstall every installed plugin, which is useful before removing
the Stripe CLI itself: package managers leave plugins in place when they
uninstall the stripe binary.`,
		RunE: uc.runUninstallCmd,
	}

	uc.Cmd.Flags().BoolVar(&uc.all, "all", false, "Uninstall every installed Stripe CLI plugin")

	return uc
}

// validateArgs requires exactly one plugin name, unless --all is set, in which
// case a plugin name must not be provided.
func (uc *UninstallCmd) validateArgs(cmd *cobra.Command, args []string) error {
	if !uc.all {
		return validators.ExactArgs(1)(cmd, args)
	}

	if len(args) > 0 {
		return errorcategory.UserInputErrorf(
			"`%s --all` does not take a plugin name. Remove `--all` to uninstall only `%s`, or remove the plugin name to uninstall every plugin",
			cmd.CommandPath(),
			args[0],
		)
	}

	return nil
}

func (uc *UninstallCmd) runUninstallCmd(cmd *cobra.Command, args []string) error {
	// GetString only errors when the flag isn't registered on cmd, which happens
	// in tests that build UninstallCmd without attaching it under the root command
	// (where --api-base/--dashboard-base are defined); fall back to the same
	// default the root flag would carry.
	apiBaseURL, err := cmd.Flags().GetString("api-base")
	if err != nil {
		apiBaseURL = stripe.DefaultAPIBaseURL
	}
	if err := stripe.ValidateAPIBaseURL(apiBaseURL); err != nil {
		return err
	}
	rawDashboardBaseURL, _ := cmd.Flags().GetString("dashboard-base")
	dashboardBaseURL := resolveDashboardBaseURL(apiBaseURL, rawDashboardBaseURL)
	if err := stripe.ValidateDashboardBaseURL(dashboardBaseURL); err != nil {
		return err
	}
	accessBaseURL, _ := cmd.Flags().GetString("access-base")

	baseURLs := hookBaseURLs{
		api:       explicitFlagValue(cmd, "api-base", apiBaseURL),
		dashboard: explicitFlagValue(cmd, "dashboard-base", rawDashboardBaseURL),
		access:    explicitFlagValue(cmd, "access-base", accessBaseURL),
	}

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.uninstallCmd.runUninstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	if uc.all {
		return uc.uninstallAllPlugins(ctx, cmd, baseURLs)
	}

	if err := plugins.ValidatePluginShortname(args[0]); err != nil {
		return err
	}

	return uc.uninstallPlugin(ctx, cmd, args[0], baseURLs)
}

// uninstallAllPlugins uninstalls every plugin recorded in the config's installed
// plugins list or recovered from local plugin metadata, stopping at the first
// plugin that cannot be removed.
func (uc *UninstallCmd) uninstallAllPlugins(ctx context.Context, cmd *cobra.Command, baseURLs hookBaseURLs) error {
	pluginNames, err := plugins.GetInstalledPluginNames(uc.cfg, uc.fs)
	if err != nil {
		return err
	}

	if len(pluginNames) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Stripe CLI plugins are installed, nothing to uninstall.")
		return nil
	}

	for _, pluginName := range pluginNames {
		if err := uc.uninstallPlugin(ctx, cmd, pluginName, baseURLs); err != nil {
			return err
		}
	}

	return nil
}

func (uc *UninstallCmd) uninstallPlugin(ctx context.Context, cmd *cobra.Command, pluginName string, baseURLs hookBaseURLs) error {
	plugin := plugins.Plugin{Shortname: pluginName}

	if m := stripe.GetEventMetadata(cmd.Context()); m != nil {
		m.SetPluginName(plugin.Shortname)
	}

	installedVersion := plugin.InstalledVersion(uc.cfg, uc.fs)

	if installedVersion != "" {
		runPreUninstallHook(ctx, uc.cfg, uc.fs, &plugin, installedVersion,
			baseURLs.api, baseURLs.dashboard, baseURLs.access)
	}

	if err := plugin.Uninstall(ctx, uc.cfg, uc.fs); err != nil {
		return err
	}

	sendPluginLifecycleEvent(cmd.Context(), "Plugin Uninstalled", installedVersion)

	out := cmd.OutOrStdout()
	color := ansi.Color(out)
	successMsg := fmt.Sprintf("✔ %s has been uninstalled.", plugin.Shortname)
	fmt.Fprintln(out, color.Green(successMsg))

	return nil
}
