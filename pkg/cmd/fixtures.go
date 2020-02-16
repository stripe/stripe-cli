package cmd

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

// FixturesCmd prints a list of all the available sample projects that users can
// generate
type FixturesCmd struct {
	Cmd *cobra.Command
	Cfg *config.Config

	stripeAccount string
}

func newFixturesCmd(cfg *config.Config) *FixturesCmd {
	fixturesCmd := &FixturesCmd{
		Cfg: cfg,
	}

	fixturesCmd.Cmd = &cobra.Command{
		Use:   "fixtures",
		Args:  validators.ExactArgs(1),
		Short: "Run fixtures to populate your account with data",
		Long:  `Run fixtures to populate your account with data`,
		RunE:  fixturesCmd.runFixturesCmd,
	}

	fixturesCmd.Cmd.Flags().StringVar(&fixturesCmd.stripeAccount, "stripe-account", "", "Set a header identifying the connected account")

	return fixturesCmd
}

func (fc *FixturesCmd) runFixturesCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	apiKey, err := fc.Cfg.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return nil
	}

	fixture, err := fixtures.NewFixture(
		afero.NewOsFs(),
		apiKey,
		fc.stripeAccount,
		stripe.DefaultAPIBaseURL,
		args[0],
	)
	if err != nil {
		return err
	}

	err = fixture.Execute()
	if err != nil {
		return err
	}

	err = fixture.UpdateEnv()
	if err != nil {
		return err
	}

	return nil
}
