package plugin

import (
	"errors"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
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
	ctx := withSIGTERMCancel(cmd.Context(), func() {
		log.WithFields(log.Fields{
			"prefix": "cmd.uninstallCmd.runUninstallCmd",
		}).Debug("Ctrl+C received, cleaning up...")
	})

	plugin, err := plugins.LookUpPlugin(cmd.Context(), uc.cfg, uc.fs, args[0])

	if err != nil {
		return errors.New("this plugin doesn't seem to exist")
	}

	err = plugin.Uninstall(ctx, uc.cfg, uc.fs)

	if err == nil {
		color := ansi.Color(os.Stdout)
		successMsg := fmt.Sprintf("✔ %s has been uninstalled.", plugin.Shortname)
		fmt.Println(color.Green(successMsg))
	}

	return err
}
