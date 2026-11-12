package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	genqlient "github.com/Khan/genqlient/graphql"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2/gqlerror"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const v2TestmodeUnsupportedCode = "v2_api_not_supported_in_testmode"

var newOverviewGraphQLClient = graphql.NewClient

type overviewCmd struct {
	cmd        *cobra.Command
	format     string
	limit      int
	livemode   bool
	apiBaseURL string
}

func newOverviewCmd() *overviewCmd {
	oc := &overviewCmd{}

	oc.cmd = &cobra.Command{
		Use:    "overview",
		Args:   validators.NoArgs,
		Hidden: true,
		Short:  "Show integration health alerts for your account",
		Long: `Show integration health alerts Stripe has detected for your account,
along with why each matters and what to do about it.

Uses the mode of your active account context. Alerts are only available in
live mode and sandboxes, so pass --live if your active context is test mode.`,
		Example: `stripe overview
  stripe overview --live
  stripe overview --format json`,
		RunE: oc.runOverviewCmd,
	}

	oc.cmd.Flags().StringVar(&oc.format, "format", "", "Output format: 'json' for the full alert details")
	oc.cmd.Flags().IntVar(&oc.limit, "limit", 0, "Maximum number of alerts to return")
	oc.cmd.Flags().BoolVar(&oc.livemode, "live", false, "Show live mode alerts (default: your active context's mode). Requires that you're logged in to live mode — check with 'stripe whoami', switch with 'stripe switch'")
	oc.cmd.Flags().StringVar(&oc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	oc.cmd.Flags().MarkHidden("api-base") //nolint:errcheck

	return oc
}

func (oc *overviewCmd) runOverviewCmd(cmd *cobra.Command, args []string) error {
	if oc.format != "" && oc.format != "json" {
		return errorcategory.UserInputErrorf("unsupported --format %q (supported: json)", oc.format)
	}
	if oc.limit < 0 {
		return errorcategory.UserInputErrorf("--limit must be a positive number")
	}

	livemode := oc.livemode
	if !livemode {
		// Follow the active context's mode rather than forcing test mode.
		creds, err := Config.GetProfile().ResolveCredentialsForAnyMode(false)
		if err != nil {
			return err
		}
		livemode = creds.Livemode()
	}

	client, err := newOverviewGraphQLClient(cmd.Context(), graphql.Options{
		Profile:       Config.GetProfile(),
		Livemode:      livemode,
		APIBaseURL:    oc.apiBaseURL,
		StripeVersion: requests.StripePreviewVersionHeaderValue,
	})
	if err != nil {
		return err
	}

	resp, err := fetchIntegHlthAlerts(cmd.Context(), client, oc.limit)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if oc.format == "json" {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	}
	renderIntegHlthAlerts(out, resp.Alerts)
	return nil
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func overviewGraphQLError(err error, resource string) error {
	var httpErr *genqlient.HTTPError
	if errors.As(err, &httpErr) {
		if httpErr.StatusCode == http.StatusUnauthorized {
			return errorcategory.Errorf(errorcategory.Auth, "credentials rejected (status 401); run `stripe login`")
		}
		return errorcategory.Errorf(errorcategory.API, "failed to retrieve %s (status %d): %s", resource, httpErr.StatusCode, gqlErrorMessages(httpErr.Response.Errors))
	}

	var gqlErrs gqlerror.List
	if errors.As(err, &gqlErrs) {
		for _, e := range gqlErrs {
			if code, _ := e.Extensions["code"].(string); code == v2TestmodeUnsupportedCode {
				return errorcategory.UserInputErrorf("%s aren't available in test mode; re-run with --live, or run `stripe switch` to select a sandbox", resource)
			}
		}
		return errorcategory.Errorf(errorcategory.API, "failed to retrieve %s: %s", resource, gqlErrorMessages(gqlErrs))
	}

	return err
}

func gqlErrorMessages(errs gqlerror.List) string {
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, strings.TrimSpace(e.Message))
	}
	return strings.Join(messages, "; ")
}

func renderIntegHlthAlerts(w io.Writer, alerts []integHlthAlert) {
	if len(alerts) == 0 {
		fmt.Fprintln(w, "No integration health alerts. Your integration looks healthy.")
		return
	}

	header := fmt.Sprintf("%d recommended actions:", len(alerts))
	if len(alerts) == 1 {
		header = "1 recommended action:"
	}
	fmt.Fprintln(w, ansi.Bold(header))
	for i, a := range alerts {
		fmt.Fprintf(w, "%d. %s. Run %s for more information.\n",
			i+1, integHlthAlertActionTitle(a), ansi.Bold("`"+integHlthAlertNextCommand(a)+"`"))
	}
}

// integHlthAlertActionTitle returns the alert's recommended action title,
// falling back to its summary and then its predicate name.
func integHlthAlertActionTitle(a integHlthAlert) string {
	title := a.PredicateName
	if insight := a.HealthInsight; insight != nil {
		switch {
		case insight.RecommendedAction != nil && insight.RecommendedAction.Title != "":
			title = insight.RecommendedAction.Title
		case insight.Summary != nil && *insight.Summary != "":
			title = *insight.Summary
		}
	}
	return strings.TrimRight(strings.TrimSpace(title), ".")
}

// integHlthAlertNextCommand returns the command that gives more detail on an
// alert: `stripe doctor` when the alert maps to a doctor topic, otherwise
// `stripe health` for the alert itself.
func integHlthAlertNextCommand(a integHlthAlert) string {
	if a.DoctorTopic != nil && *a.DoctorTopic != "" {
		return "stripe doctor " + *a.DoctorTopic
	}
	return "stripe health " + a.HealthInsightID
}
