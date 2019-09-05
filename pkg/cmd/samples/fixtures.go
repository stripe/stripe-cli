package samples

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/config"
	s "github.com/stripe/stripe-cli/pkg/samples"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// FixturesCmd prints a list of all the available sample projects that users can
// generate
type FixturesCmd struct {
	Cmd *cobra.Command
	Cfg *config.Config
}

// NewFixturesCmd creates and returns a list command for samples
func NewFixturesCmd(cfg *config.Config) *FixturesCmd {
	fixturesCmd := &FixturesCmd{
		Cfg: cfg,
	}

	fixturesCmd.Cmd = &cobra.Command{
		Use:   "fixtures",
		Short: "Run fixtures to populate your account with data",
		Long:  `Run fixtures to populate your account with data`,
		RunE:  fixturesCmd.runFixturesCmd,
	}

	return fixturesCmd
}

func (fc *FixturesCmd) runFixturesCmd(cmd *cobra.Command, args []string) error {
	apiKey, err := fc.Cfg.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	fixture, err := s.NewFixture(
		afero.NewOsFs(),
		apiKey,
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
