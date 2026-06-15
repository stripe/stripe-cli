package plugin

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// ListCmd is the struct used for configuring the plugin list command.
type ListCmd struct {
	cfg              *config.Config
	Cmd              *cobra.Command
	listPlugins      func(context.Context, config.IConfig, string, string) (plugins.PluginList, error)
	apiBaseURL       string
	dashboardBaseURL string
}

// NewListCmd creates a command for listing available plugins.
func NewListCmd(config *config.Config) *ListCmd {
	lc := &ListCmd{}
	lc.cfg = config
	lc.listPlugins = plugins.ListPlugins

	lc.Cmd = &cobra.Command{
		Use:   "list",
		Args:  validators.NoArgs,
		Short: "List available Stripe CLI plugins",
		Long:  "List available Stripe CLI plugins.",
		RunE:  lc.runListCmd,
	}

	// Hidden configuration flags, useful for dev/debugging
	lc.Cmd.Flags().StringVar(&lc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	lc.Cmd.Flags().MarkHidden("api-base") // #nosec G104
	lc.Cmd.Flags().StringVar(&lc.dashboardBaseURL, "dashboard-base", "", "Sets the dashboard base URL")
	lc.Cmd.Flags().MarkHidden("dashboard-base") // #nosec G104

	return lc
}

func (lc *ListCmd) runListCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(lc.apiBaseURL); err != nil {
		return err
	}
	dashboardBaseURL := resolveDashboardBaseURL(lc.apiBaseURL, lc.dashboardBaseURL)
	if err := stripe.ValidateDashboardBaseURL(dashboardBaseURL); err != nil {
		return err
	}

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	pluginList, err := lc.listPlugins(ctx, lc.cfg, lc.apiBaseURL, dashboardBaseURL)
	if err != nil {
		return err
	}

	availablePlugins := make([]plugins.Plugin, 0, len(pluginList.Plugins))
	nameWidth := 0
	for _, plugin := range pluginList.Plugins {
		if plugin.Shortname == "" || plugin.LookUpLatestVersion() == "" {
			continue
		}
		availablePlugins = append(availablePlugins, plugin)
		if len(plugin.Shortname) > nameWidth {
			nameWidth = len(plugin.Shortname)
		}
	}

	sort.Slice(availablePlugins, func(i, j int) bool {
		return availablePlugins[i].Shortname < availablePlugins[j].Shortname
	})

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Available Stripe plugins:")

	if len(availablePlugins) == 0 {
		fmt.Fprintln(out, "  No plugins are currently available for this platform.")
		return nil
	}

	for _, plugin := range availablePlugins {
		if plugin.Shortdesc == "" {
			fmt.Fprintf(out, "  %s\n", plugin.Shortname)
			continue
		}
		fmt.Fprintf(out, "  %-*s  %s\n", nameWidth, plugin.Shortname, plugin.Shortdesc)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Run `stripe plugin install <name>` to install a plugin.")

	return nil
}
