// Package plugin provides plugin management commands.
package plugin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	goversion "github.com/hashicorp/go-version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// InstallCmd is the struct used for configuring the plugin install command
type InstallCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
	fs  afero.Fs

	apiBaseURL string
}

// NewInstallCmd creates a command for installing plugins
func NewInstallCmd(config *config.Config) *InstallCmd {
	ic := &InstallCmd{}
	ic.fs = afero.NewOsFs()
	ic.cfg = config

	ic.Cmd = &cobra.Command{
		Use:   "install",
		Args:  validators.ExactArgs(1),
		Short: "Install a Stripe CLI plugin",
		Long: `Install a Stripe CLI plugin. To download a specific version, run stripe install [plugin_name]@[version].
			By default, the most recent version will be installed.`,
		RunE: ic.runInstallCmd,
	}

	// Hidden configuration flags, useful for dev/debugging
	ic.Cmd.Flags().StringVar(&ic.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	ic.Cmd.Flags().MarkHidden("api-base") // #nosec G104

	return ic
}

// parsePluginArg takes in the argument and returns the name of the plugin to download and the version
// Ex: parseArg('plugin') -> 'plugin', nil
// Ex: parseArg('plugin@2.0.2') -> 'plugin', '2.0.2'
func parseInstallArg(arg string) (string, string) {
	args := strings.Split(arg, "@")
	plugin := args[0]
	version := ""
	if len(args) > 1 {
		version = args[1]
	}

	return plugin, version
}

func (ic *InstallCmd) runInstallCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(ic.apiBaseURL); err != nil {
		return err
	}

	color := ansi.Color(os.Stdout)
	pluginName, version := parseInstallArg(args[0])
	ic.setInstallTelemetryMetadata(cmd.Context(), pluginName)

	// Refresh the plugin before proceeding
	if err := plugins.RefreshPluginManifest(cmd.Context(), ic.cfg, ic.fs, ic.apiBaseURL); err != nil {
		return err
	}

	plugin, err := plugins.LookUpPlugin(cmd.Context(), ic.cfg, ic.fs, pluginName)
	if err != nil {
		return err
	}

	isLatest := len(version) == 0
	if isLatest {
		version = plugin.LookUpLatestVersion()
	}

	if plugin.IsVersionInstalled(ic.cfg, ic.fs, version) {
		if isLatest {
			fmt.Println(color.Green(fmt.Sprintf("✔ v%s is already installed (latest).", version)))
		} else {
			fmt.Println(color.Green(fmt.Sprintf("✔ v%s is already installed.", version)))
		}
		return nil
	}

	prevVersion := plugin.InstalledVersion(ic.cfg, ic.fs)

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.installCmd.runInstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	if err := plugin.Install(ctx, ic.cfg, ic.fs, version, ic.apiBaseURL); err != nil {
		return err
	}

	if prevVersion != "" {
		fmt.Println(color.Green(fmt.Sprintf("✔ %s from v%s to v%s.", versionChangeVerb(prevVersion, version), prevVersion, version)))
	} else {
		fmt.Println(color.Green(fmt.Sprintf("✔ installation of v%s complete.", version)))
	}

	return nil
}

func versionChangeVerb(from, to string) string {
	prev, prevErr := goversion.NewVersion(from)
	next, nextErr := goversion.NewVersion(to)
	if prevErr == nil && nextErr == nil && prev.GreaterThan(next) {
		return "downgraded"
	}
	return "upgraded"
}

func (ic *InstallCmd) setInstallTelemetryMetadata(ctx context.Context, pluginName string) {
	telemetryMetadata := stripe.GetEventMetadata(ctx)
	if telemetryMetadata != nil {
		telemetryMetadata.SetPluginName(pluginName)
	}
}

func withSIGTERMCancel(ctx context.Context, onCancel func()) context.Context {
	// Create a context that will be canceled when Ctrl+C is pressed
	ctx, cancel := context.WithCancel(ctx)

	interruptCh := make(chan os.Signal, 1)
	signal.Notify(interruptCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-interruptCh
		onCancel()
		cancel()
	}()
	return ctx
}
