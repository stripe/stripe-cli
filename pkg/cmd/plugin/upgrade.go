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

type upgradeCmd struct {
	cfg *config.Config
	Cmd *cobra.Command
	fs  afero.Fs
}

func NewUpgradeCmd(config *config.Config) *upgradeCmd {
	uc := &upgradeCmd{}
	uc.fs = afero.NewOsFs()
	uc.cfg = config

	uc.Cmd = &cobra.Command{
		Use:   "upgrade",
		Args:  validators.ExactArgs(1),
		Short: "Upgrade a Stripe CLI plugin",
		Long:  "Upgrade a Stripe CLI plugin to the latest version available. To download a specific version, please see the `install` command",
		RunE:  uc.runUpgradeCmd,
	}

	return uc
}

func (uc *upgradeCmd) runUpgradeCmd(cmd *cobra.Command, args []string) error {
	// Refresh the plugin before proceeding
	plugins.RefreshPluginManifest(cmd.Context(), uc.cfg, uc.fs, stripe.DefaultAPIBaseURL)

	plugin, err := plugins.LookUpPlugin(cmd.Context(), uc.cfg, uc.fs, args[0])
	if err != nil {
		return err
	}
	version := plugin.LookUpLatestVersion()

	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.upgradeCmd.runUpgradeCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	err = plugin.Install(ctx, uc.cfg, uc.fs, version, stripe.DefaultAPIBaseURL)

	if err == nil {
		color := ansi.Color(os.Stdout)
		fmt.Println(color.Green("✔ upgrade complete."))
	}

	return err
}
