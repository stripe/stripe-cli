package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/validators"
)

type loginCmd struct {
	cmd         *cobra.Command
	interactive bool
	url         string
}

func newLoginCmd() *loginCmd {
	lc := &loginCmd{}

	lc.cmd = &cobra.Command{
		Use:   "login",
		Args:  validators.NoArgs,
		Short: "Log into your Stripe account",
		Long:  `Log into your Stripe account to write your configuration file`,
		RunE:  lc.runLoginCmd,
	}
	lc.cmd.Flags().BoolVarP(&lc.interactive, "interactive", "i", false, "interactive configuration mode")
	lc.cmd.Flags().StringVarP(&lc.url, "url", "u", "", "Testing URL for login ")
	lc.cmd.Flags().MarkHidden("url")

	return lc
}

func (lc *loginCmd) runLoginCmd(cmd *cobra.Command, args []string) error {
	if lc.interactive {
		return login.InteractiveLogin(Profile)
	}
	return login.Login(lc.url, Profile, os.Stdin)
}
