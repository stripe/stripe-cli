package plugin

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// UninstallCmd is the struct used for configuring the plugin uninstall command
type UninstallCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
	fs  afero.Fs
}

// NewUninstallCmd creates a new command for uninstalling plugins
func NewUninstallCmd(config *config.Config) *UninstallCmd {
	uc := &UninstallCmd{}
	uc.fs = afero.NewOsFs()
	uc.cfg = config

	uc.Cmd = &cobra.Command{
		Use:   "uninstall",
		Args:  validators.ExactArgs(1),
		Short: "Uninstall a Stripe CLI plugin",
		Long:  "Uninstall a Stripe CLI plugin.",
		RunE:  uc.runUninstallCmd,
	}

	return uc
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

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.uninstallCmd.runUninstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	if err := plugins.ValidatePluginShortname(args[0]); err != nil {
		return err
	}

	plugin := plugins.Plugin{Shortname: args[0]}

	if m := stripe.GetEventMetadata(cmd.Context()); m != nil {
		m.SetPluginName(plugin.Shortname)
	}

	installedVersion := plugin.InstalledVersion(uc.cfg, uc.fs)

	if installedVersion != "" {
		accessBaseURL, _ := cmd.Flags().GetString("access-base")
		runPreUninstallHook(ctx, uc.cfg, uc.fs, &plugin, installedVersion,
			explicitFlagValue(cmd, "api-base", apiBaseURL),
			explicitFlagValue(cmd, "dashboard-base", rawDashboardBaseURL),
			explicitFlagValue(cmd, "access-base", accessBaseURL))
	}

	uninstallErr := plugin.Uninstall(ctx, uc.cfg, uc.fs)

	if uninstallErr == nil {
		sendPluginLifecycleEvent(cmd.Context(), "Plugin Uninstalled", installedVersion)
		color := ansi.Color(os.Stdout)
		successMsg := fmt.Sprintf("✔ %s has been uninstalled.", plugin.Shortname)
		fmt.Println(color.Green(successMsg))
	}

	return uninstallErr
}
