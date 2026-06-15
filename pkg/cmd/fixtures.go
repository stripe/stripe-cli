package cmd

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/i18n"
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
	apiVersion    string
	apiBaseURL    string
	skip          []string
	override      []string
	add           []string
	remove        []string
	edit          bool
}

func newFixturesCmd(cfg *config.Config) *FixturesCmd {
	fixturesCmd := &FixturesCmd{
		Cfg: cfg,
	}

	fixturesCmd.Cmd = &cobra.Command{
		Use:   "fixtures",
		Args:  validators.ExactArgs(1),
		Short: i18n.T("fixtures.short"),
		Long:  i18n.T("fixtures.long"),
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Fixtures execute a sequence of API requests defined in a JSON file.\n" +
				"  Use `--override` to customize parameters, e.g. `--override customer:email=test@example.com`.\n" +
				"  Reference prior responses with `${resource:json_path}` and env vars with `${.env:VAR|default}`.",
		},
		RunE: fixturesCmd.runFixturesCmd,
	}

	fixturesCmd.Cmd.Flags().StringVar(&fixturesCmd.stripeAccount, "stripe-account", "", i18n.T("fixtures.flags.stripe_account"))
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.skip, "skip", []string{}, i18n.T("fixtures.flags.skip"))
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.override, "override", []string{}, i18n.T("fixtures.flags.override"))
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.add, "add", []string{}, i18n.T("fixtures.flags.add"))
	fixturesCmd.Cmd.Flags().StringArrayVar(&fixturesCmd.remove, "remove", []string{}, i18n.T("fixtures.flags.remove"))
	fixturesCmd.Cmd.Flags().StringVar(&fixturesCmd.apiVersion, "api-version", "", i18n.T("fixtures.flags.api_version"))
	fixturesCmd.Cmd.Flags().BoolVar(&fixturesCmd.edit, "edit", false, i18n.T("fixtures.flags.edit"))

	// Hidden configuration flags, useful for dev/debugging
	fixturesCmd.Cmd.Flags().StringVar(&fixturesCmd.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	fixturesCmd.Cmd.Flags().MarkHidden("api-base") // #nosec G104

	return fixturesCmd
}

func (fc *FixturesCmd) runFixturesCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	if err := stripe.ValidateAPIBaseURL(fc.apiBaseURL); err != nil {
		return err
	}

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
		fc.apiBaseURL,
		args[0],
		fc.skip,
		fc.override,
		fc.add,
		fc.remove,
		fc.edit,
	)
	if err != nil {
		return err
	}

	_, err = fixture.Execute(cmd.Context(), fc.apiVersion)

	if err != nil {
		return err
	}

	err = fixture.UpdateEnv()
	if err != nil {
		return err
	}

	return nil
}
