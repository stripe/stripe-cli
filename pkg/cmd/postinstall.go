package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type postinstallCmd struct {
	cfg *config.Config
	cmd *cobra.Command
}

func newPostinstallCmd(config *config.Config) *postinstallCmd {
	pic := &postinstallCmd{
		cfg: config,
	}
	pic.cmd = &cobra.Command{
		Use:     "postinstall",
		Args:    validators.NoArgs,
		Short:   "Run some checks after installation of this CLI and prompt user if needed",
		Example: `stripe postinstall`,
		Hidden:  true,
		RunE:    pic.runPostinstallCmd,
	}
	return pic
}

func (pic *postinstallCmd) runPostinstallCmd(cmd *cobra.Command, args []string) error {
	color := ansi.Color(os.Stdout)
	_, err := pic.cfg.Profile.GetAPIKey(false)

	// If we can't get the API key, then it's likely that this is a first install rather than an upgrade.
	// Suggest the user run `stripe login` to get started as a helpful prompt.
	if err != nil {
		welcomeIcon := color.BrightRed("❤").String()
		welcomeText := "Thanks for installing the Stripe CLI! To get started, run `stripe login`"
		fmt.Printf("%s %s\n", welcomeIcon, welcomeText)
	}

	return nil
}
