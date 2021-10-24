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
	skip          []string
	override      []string
	add           []string
	remove        []string
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
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.skip, "skip", []string{}, "Skip specific steps in the fixture")
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.override, "override", []string{}, "Override parameters in the fixture")
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.add, "add", []string{}, "Add parameters in the fixture")
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.remove, "remove", []string{}, "Remove parameters from the fixture")

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

	fixture, err := fixtures.NewFixtureFromFile(
		afero.NewOsFs(),
		apiKey,
		fc.stripeAccount,
		stripe.DefaultAPIBaseURL,
		args[0],
		fc.skip,
		fc.override,
		fc.add,
		fc.remove,
	)
	if err != nil {
		return err
	}

	_, err = fixture.Execute(cmd.Context())

	if err != nil {
		return err
	}

	err = fixture.UpdateEnv()
	if err != nil {
		return err
	}

	return nil
}
