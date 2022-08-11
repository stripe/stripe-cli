package plugin

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

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

	archiveURL  string
	archivePath string
}

// NewInstallCmd creates a command for installing plugins
func NewInstallCmd(config *config.Config) *InstallCmd {
	ic := &InstallCmd{}
	ic.fs = afero.NewOsFs()
	ic.cfg = config

	ic.Cmd = &cobra.Command{
		Use:   "install",
		Args:  validators.MaximumNArgs(1),
		Short: "Install a Stripe CLI plugin",
		Long: `Install a Stripe CLI plugin. To download a specific version, run stripe install [plugin_name]@[version].
			By default, the most recent version will be installed.`,
		RunE: ic.runInstallCmd,
	}

	ic.Cmd.Flags().StringVar(&ic.archiveURL, "archive-url", "", "Install a plugin by an archive URL")
	ic.Cmd.Flags().StringVar(&ic.archivePath, "archive", "", "Install a plugin by an archive path")

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

func (ic *InstallCmd) installPluginByName(cmd *cobra.Command, arg string) error {
	pluginName, version := parseInstallArg(arg)
	plugin, err := plugins.LookUpPlugin(cmd.Context(), ic.cfg, ic.fs, pluginName)

	if err != nil {
		return err
	}

	if len(version) == 0 {
		version = plugin.LookUpLatestVersion()
	}

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.installCmd.runInstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	err = plugin.Install(ctx, ic.cfg, ic.fs, version, stripe.DefaultAPIBaseURL)

	return err
}

func (ic *InstallCmd) installPluginByArchive(cmd *cobra.Command) error {
	if ic.archiveURL == "" && ic.archivePath == "" {
		return fmt.Errorf("please provide the plugin name or the archive URL/path to install")
	}

	if ic.archiveURL != "" {
		err := plugins.FetchAndExtractRemoteArchive(cmd.Context(), ic.cfg, ic.archiveURL)
		if err != nil {
			return err
		}
	} else if ic.archivePath != "" {
		err := plugins.ExtractLocalArchive(cmd.Context(), ic.cfg, ic.archivePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ic *InstallCmd) runInstallCmd(cmd *cobra.Command, args []string) error {
	var err error

	if len(args) == 0 {
		err = ic.installPluginByArchive(cmd)
	} else {
		// Refresh the plugin before proceeding
		err = plugins.RefreshPluginManifest(cmd.Context(), ic.cfg, ic.fs, stripe.DefaultAPIBaseURL)
		if err != nil {
			return err
		}

		err = ic.installPluginByName(cmd, args[0])
		if err != nil {
			return err
		}
	}

	if err == nil {
		color := ansi.Color(os.Stdout)
		fmt.Println(color.Green("✔ installation complete."))
	}

	return nil
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
