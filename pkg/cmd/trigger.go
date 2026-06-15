package cmd

import (
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/fixtures"
	"github.com/stripe/stripe-cli/pkg/i18n"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type triggerCmd struct {
	cmd *cobra.Command

	fs            afero.Fs
	stripeAccount string
	apiVersion    string
	skip          []string
	override      []string
	add           []string
	remove        []string
	raw           string
	apiBaseURL    string
	edit          bool
}

func newTriggerCmd() *triggerCmd {
	tc := &triggerCmd{}
	tc.fs = afero.NewOsFs()
	tc.cmd = &cobra.Command{
		Use:       "trigger <event>",
		Args:      validators.MaximumNArgs(1),
		ValidArgs: fixtures.EventNames(),
		Short:     i18n.T("trigger.short"),
		Long: i18n.Tf("trigger.long",
			i18n.Args{
				"supported_events_header": ansi.Bold("Supported events:"),
				"event_list":              fixtures.EventList(),
			},
		),
		Example: i18n.T("trigger.example"),
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use `--override` to customize event data, e.g. `--override customer:email=test@example.com`.\n" +
				"  Use `--skip` to skip specific steps in the trigger sequence.\n" +
				"  Triggers create real API objects in test mode that you can inspect afterward.",
		},
		RunE: tc.runTriggerCmd,
	}

	tc.cmd.Flags().StringVar(&tc.stripeAccount, "stripe-account", "", i18n.T("trigger.flags.stripe_account"))
	tc.cmd.Flags().StringArrayVar(&tc.skip, "skip", []string{}, i18n.T("trigger.flags.skip"))
	tc.cmd.Flags().StringArrayVar(&tc.override, "override", []string{}, i18n.T("trigger.flags.override"))
	tc.cmd.Flags().StringArrayVar(&tc.add, "add", []string{}, i18n.T("trigger.flags.add"))
	tc.cmd.Flags().StringArrayVar(&tc.remove, "remove", []string{}, i18n.T("trigger.flags.remove"))
	tc.cmd.Flags().StringVar(&tc.raw, "raw", "", i18n.T("trigger.flags.raw"))
	tc.cmd.Flags().StringVar(&tc.apiVersion, "api-version", "", i18n.T("trigger.flags.api_version"))
	tc.cmd.Flags().BoolVar(&tc.edit, "edit", false, i18n.T("trigger.flags.edit"))

	// Hidden configuration flags, useful for dev/debugging
	tc.cmd.Flags().StringVar(&tc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	tc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return tc
}

func (tc *triggerCmd) runTriggerCmd(cmd *cobra.Command, args []string) error {
	version.CheckLatestVersion()

	if err := stripe.ValidateAPIBaseURL(tc.apiBaseURL); err != nil {
		return err
	}

	if len(args) == 0 {
		cmd.Help()

		return nil
	}

	apiKey, err := Config.Profile.GetAPIKey(false)
	if err != nil {
		return err
	}

	event := args[0]

	_, err = fixtures.Trigger(cmd.Context(), event, tc.stripeAccount, tc.apiBaseURL, apiKey, tc.skip, tc.override, tc.add, tc.remove, tc.raw, tc.apiVersion, tc.edit)
	if err != nil {
		return err
	}

	fmt.Println(i18n.T("trigger.output.success"))
	return nil
}
