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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var (
	newOverviewGraphQLClient   = graphql.NewClient
	resolveOverviewCredentials = activeOverviewCredentials
)

// Error messages `stripe overview` shows. Errors the reader can't act on are
// shown as overviewUnavailableMessage, with the cause logged at debug level.
const (
	// overviewUnsupportedFormatMessage takes the --format value.
	overviewUnsupportedFormatMessage        = "unsupported --format %q (supported: json)"
	overviewInvalidLimitMessage             = "--limit must be greater than 0"
	overviewLiveFromSandboxMessage          = "You're in a sandbox. Run `stripe overview --force-live` to see insights for your live account."
	overviewLiveAccountNotAuthorizedMessage = "Your live account isn't authorized. Run `stripe login` to authorize it, then run `stripe overview --force-live`."
	overviewLiveAccountLookupFailedMessage  = "We couldn't find the live account for your sandbox right now. Please try again later."
	overviewCredentialsRejectedMessage      = "Your credentials were rejected. Run `stripe login` to log in again."
	overviewUnavailableMessage              = "We're having trouble loading your integration health insights right now. Please try again later."
)

// overviewJSON is the --format json output. It carries what the text output
// shows, in a stable shape for scripts and agents.
type overviewJSON struct {
	Account            overviewJSONAccount  `json:"account"`
	DefaultAPIVersion  string               `json:"default_api_version,omitempty"`
	RecommendedActions []overviewJSONAction `json:"recommended_actions"`
	Notice             *overviewJSONNotice  `json:"notice,omitempty"`
}

type overviewJSONAccount struct {
	ID string `json:"id,omitempty"`
	// Mode is "live", "sandbox", or for API keys, "test".
	Mode string `json:"mode"`
}

type overviewJSONAction struct {
	Title string            `json:"title"`
	Guide overviewJSONGuide `json:"guide"`
}

type overviewJSONGuide struct {
	Command string `json:"command"`
}

type overviewJSONNotice struct {
	Message   string             `json:"message"`
	NextSteps []overviewJSONStep `json:"next_steps"`
}

type overviewJSONStep struct {
	Command string `json:"command"`
	Purpose string `json:"purpose"`
}

func newOverviewJSON(target overviewTarget, version string, alerts []integHlthAlert) overviewJSON {
	out := overviewJSON{
		Account:            overviewJSONAccount{ID: target.creds.OAKContext, Mode: overviewAccountMode(target.creds)},
		DefaultAPIVersion:  version,
		RecommendedActions: make([]overviewJSONAction, 0, len(alerts)),
		Notice:             sandboxNotice(target.footer),
	}
	for _, a := range alerts {
		out.RecommendedActions = append(out.RecommendedActions, overviewJSONAction{
			Title: integHlthAlertActionTitle(a),
			Guide: overviewJSONGuide{Command: integHlthAlertDocsCommand(a)},
		})
	}
	return out
}

func overviewAccountMode(creds stripe.Credentials) string {
	switch {
	case creds.Livemode():
		return "live"
	case creds.OAKContext != "":
		return "sandbox"
	default:
		return "test"
	}
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

	oc.cmd.Flags().StringVar(&oc.format, "format", "", "Output format: 'json' for a stable JSON schema (suitable for scripts and agents)")
	oc.cmd.Flags().IntVar(&oc.limit, "limit", 0, "Maximum number of alerts to return (default: all)")
	oc.cmd.Flags().BoolVar(&oc.livemode, "live", false, "Show live mode alerts (default: your active context's mode)")
	oc.cmd.Flags().BoolVar(&oc.forceLive, "force-live", false, "In a sandbox, show insights for the live account that owns it")
	oc.cmd.Flags().StringVar(&oc.apiBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	oc.cmd.Flags().MarkHidden("api-base") //nolint:errcheck

	return oc
}

func (oc *overviewCmd) runOverviewCmd(cmd *cobra.Command, args []string) error {
	if oc.format != "" && oc.format != "json" {
		return errorcategory.UserInputErrorf(overviewUnsupportedFormatMessage, oc.format)
	}
	// Leaving --limit unset returns every alert; an explicit value must be at least 1.
	if cmd.Flags().Changed("limit") && oc.limit <= 0 {
		return errorcategory.New(errorcategory.UserInput, overviewInvalidLimitMessage)
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
		return overviewUnavailableError(err)
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
		return enc.Encode(newOverviewJSON(target, version, resp.Alerts))
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
			return overviewTarget{}, errorcategory.New(errorcategory.UserInput, overviewLiveFromSandboxMessage)
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
		return overviewTarget{}, errorcategory.New(errorcategory.API, overviewLiveAccountLookupFailedMessage)
	}
	if parentID == "" {
		return overviewTarget{}, errorcategory.New(errorcategory.UserInput, overviewLiveAccountNotAuthorizedMessage)
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

func overviewUnavailableError(err error) error {
	log.Debugf("stripe overview: %v", err)
	return errorcategory.New(errorcategory.API, overviewUnavailableMessage)
}

func overviewGraphQLError(err error) error {
	var httpErr *genqlient.HTTPError
	if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusUnauthorized {
		log.Debugf("stripe overview: %v", err)
		return errorcategory.New(errorcategory.Auth, overviewCredentialsRejectedMessage)
	}

	return overviewUnavailableError(err)
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
	return "Run " + ansi.Bold("`"+integHlthAlertDocsCommand(a)+"`") + " and follow the guide to make this change."
}

func integHlthAlertDocsCommand(a integHlthAlert) string {
	path, _ := integHlthAlertDocsPath(a)
	return "stripe docs " + path
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
