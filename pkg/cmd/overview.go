package cmd

import (
	"context"
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
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const v2TestmodeUnsupportedCode = "v2_api_not_supported_in_testmode"

var (
	newOverviewGraphQLClient   = graphql.NewClient
	resolveOverviewCredentials = activeOverviewCredentials
)

// overviewJSON is the --format json output: the account's default API version
// alongside the integration health alerts.
type overviewJSON struct {
	DefaultAPIVersion string `json:"default_api_version,omitempty"`
	*integHlthAlertsResponse
}

type overviewCmd struct {
	cmd        *cobra.Command
	format     string
	limit      int
	livemode   bool
	forceLive  bool
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

Uses the mode of your active account context. If you're in a sandbox, run
'stripe overview --force-live' to see insights for your live account.`,
		Example: `stripe overview
  stripe overview --force-live
  stripe overview --format json`,
		RunE: oc.runOverviewCmd,
	}

	oc.cmd.Flags().StringVar(&oc.format, "format", "", "Output format: 'json' for the full alert details")
	oc.cmd.Flags().IntVar(&oc.limit, "limit", 0, "Maximum number of alerts to return (default: all)")
	oc.cmd.Flags().BoolVar(&oc.livemode, "live", false, "Show live mode alerts (default: your active context's mode)")
	oc.cmd.Flags().BoolVar(&oc.forceLive, "force-live", false, "In a sandbox, show insights for the live account that owns it")
	oc.cmd.Flags().StringVar(&oc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	oc.cmd.Flags().MarkHidden("api-base") //nolint:errcheck

	return oc
}

func (oc *overviewCmd) runOverviewCmd(cmd *cobra.Command, args []string) error {
	if oc.format != "" && oc.format != "json" {
		return errorcategory.UserInputErrorf("unsupported --format %q (supported: json)", oc.format)
	}
	// Leaving --limit unset returns every alert; an explicit value must be at least 1.
	if cmd.Flags().Changed("limit") && oc.limit <= 0 {
		return errorcategory.UserInputErrorf("--limit must be greater than 0")
	}

	profile := Config.GetProfile()
	target, err := oc.resolveOverviewTarget(cmd.Context(), profile)
	if err != nil {
		return err
	}

	versionCh := startOverviewDefaultAPIVersion(cmd.Context(), profile, oc.apiBaseURL, target.creds)

	client, err := newOverviewGraphQLClient(cmd.Context(), graphql.Options{
		Profile:       profile,
		Livemode:      target.creds.Livemode(),
		Credentials:   &target.creds,
		APIBaseURL:    oc.apiBaseURL,
		StripeVersion: requests.StripePreviewVersionHeaderValue,
	})
	if err != nil {
		return err
	}

	resp, err := fetchIntegHlthAlerts(cmd.Context(), client, oc.limit)
	if err == nil {
		addIntegHlthAlertDoctorDocs(cmd.Context(), client, resp.Alerts)
		resp.Alerts = actionableIntegHlthAlerts(resp.Alerts)
	}
	version := <-versionCh

	out := cmd.OutOrStdout()
	if oc.format == "json" {
		if err != nil {
			return err
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(overviewJSON{DefaultAPIVersion: version, integHlthAlertsResponse: resp})
	}

	renderForcedLiveHeader(out, target)
	renderDefaultAPIVersion(out, version)
	if err != nil {
		renderSandboxFooter(out, target.footer)
		return err
	}
	renderIntegHlthAlerts(out, resp.Alerts)
	renderSandboxFooter(out, target.footer)
	return nil
}

// overviewTarget is the account `stripe overview` reads insights for.
type overviewTarget struct {
	creds      stripe.Credentials
	footer     sandboxFooter
	forcedLive bool
}

// resolveOverviewTarget picks the credentials to read insights with: the
// active account's, or with --force-live in a sandbox, its live account's.
func (oc *overviewCmd) resolveOverviewTarget(ctx context.Context, profile *config.Profile) (overviewTarget, error) {
	creds, err := resolveOverviewCredentials(profile, oc.livemode && !oc.forceLive)
	if err != nil {
		var mismatch *config.ActiveContextLivemodeMismatchError
		if errors.As(err, &mismatch) && !mismatch.ActiveLivemode {
			return overviewTarget{}, errorcategory.UserInputErrorf("You're in a sandbox. Run `stripe overview --force-live` to see insights for your live account.")
		}
		return overviewTarget{}, err
	}

	// Only OAuth sandbox contexts have a live account to point to.
	if creds.OAKContext == "" || creds.Livemode() {
		return overviewTarget{creds: creds}, nil
	}

	parentID, ok := lookupOverviewSandboxParent(ctx, oc.apiBaseURL, profile, creds)
	if !oc.forceLive {
		if ok && parentID == "" {
			return overviewTarget{creds: creds, footer: sandboxFooterLogin}, nil
		}
		// If the lookup failed, still point to --force-live, which reports
		// whether the live account needs authorizing.
		return overviewTarget{creds: creds, footer: sandboxFooterForceLive}, nil
	}

	if !ok {
		return overviewTarget{}, errorcategory.Errorf(errorcategory.API, "could not find the live account for your sandbox; try again")
	}
	if parentID == "" {
		return overviewTarget{}, errorcategory.UserInputErrorf("Your live account isn't authorized. Run `stripe login` to authorize it, then run `stripe overview --force-live`.")
	}
	live := stripe.NewOAKCredentials(creds.Token, parentID, true)
	return overviewTarget{creds: live, forcedLive: true}, nil
}

func activeOverviewCredentials(profile *config.Profile, live bool) (stripe.Credentials, error) {
	if live {
		return profile.ResolveCredentials(true)
	}
	return profile.ResolveCredentialsForAnyMode(false)
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
				return errorcategory.UserInputErrorf("%s aren't available in test mode", resource)
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
		fmt.Fprintln(w, "There are no recommended actions for your integration right now.")
		return
	}

	header := fmt.Sprintf("We recommend these %d actions to keep your integration healthy:", len(alerts))
	if len(alerts) == 1 {
		header = "We recommend this action to keep your integration healthy:"
	}
	fmt.Fprintln(w, ansi.Bold(header))
	for i, a := range alerts {
		fmt.Fprintf(w, "%d. %s. %s\n", i+1, integHlthAlertActionTitle(a), integHlthAlertNextStep(a))
	}
}

// integHlthAlertActionTitle returns the alert's recommended action title,
// falling back to its summary, or "" when it has neither.
func integHlthAlertActionTitle(a integHlthAlert) string {
	var title string
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

// actionableIntegHlthAlerts keeps the alerts that `stripe overview` can
// describe and point to a fix for: those with a recommended action title or
// summary, and a doctor doc on docs.stripe.com, which `stripe docs` opens.
// Other alerts are left out.
func actionableIntegHlthAlerts(alerts []integHlthAlert) []integHlthAlert {
	actionable := make([]integHlthAlert, 0, len(alerts))
	for _, a := range alerts {
		if integHlthAlertActionTitle(a) == "" {
			continue
		}
		if _, ok := integHlthAlertDocsPath(a); !ok {
			continue
		}
		actionable = append(actionable, a)
	}
	return actionable
}

// integHlthAlertDocsPath returns the `stripe docs` path for an alert's doctor
// doc, if it has one on docs.stripe.com.
func integHlthAlertDocsPath(a integHlthAlert) (string, bool) {
	if a.DoctorDoc == nil {
		return "", false
	}
	return stripeDocsPath(*a.DoctorDoc)
}

// integHlthAlertNextStep tells the reader how to act on an actionable alert.
func integHlthAlertNextStep(a integHlthAlert) string {
	// TODO: point to `stripe doctor <topic>` once the command exists.
	// if a.DoctorTopic != nil && *a.DoctorTopic != "" {
	// 	return "Run " + ansi.Bold("`stripe doctor "+*a.DoctorTopic+"`") + " and follow its steps to make this change."
	// }
	path, _ := integHlthAlertDocsPath(a)
	return "Run " + ansi.Bold("`stripe docs "+path+"`") + " and follow the guide to make this change."
}

// stripeDocsPath returns the path of a docs.stripe.com URL, which `stripe
// docs` accepts directly.
func stripeDocsPath(rawURL string) (string, bool) {
	const docsOrigin = "https://docs.stripe.com"
	if rawURL != docsOrigin && !strings.HasPrefix(rawURL, docsOrigin+"/") {
		return "", false
	}
	path := strings.TrimPrefix(rawURL, docsOrigin)
	if path == "" {
		path = "/"
	}
	return path, true
}
