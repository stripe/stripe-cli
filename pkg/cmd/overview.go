package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	integHlthAlertsPath       = "/v2/integ_hlth/alerts"
	integHlthAPIVersion       = "2026-10-12-16.internal"
	v2TestmodeUnsupportedCode = "v2_api_not_supported_in_testmode"
)

type overviewCmd struct {
	cmd           *cobra.Command
	format        string
	limit         int
	livemode      bool
	stripeAccount string
	apiBaseURL    string
}

type integHlthAlertsResponse struct {
	Alerts      []integHlthAlert `json:"alerts"`
	NextPageURL *string          `json:"next_page_url,omitempty"`
}

type integHlthAlert struct {
	HealthInsightID       string                 `json:"health_insight_id"`
	PredicateName         string                 `json:"predicate_name"`
	CompartmentID         string                 `json:"compartment_id"`
	RecentIntegrationData []string               `json:"recent_integration_data,omitempty"`
	CreatedAt             string                 `json:"created_at"`
	LastUpdated           string                 `json:"last_updated"`
	HealthInsight         *integHlthAlertInsight `json:"health_insight,omitempty"`
	Status                string                 `json:"status"`
	DoctorTopic           *string                `json:"doctor_topic,omitempty"`
}

type integHlthAlertInsight struct {
	Summary           *string                     `json:"summary,omitempty"`
	Severity          *string                     `json:"severity,omitempty"`
	WhyIsItImportant  *string                     `json:"why_is_it_important,omitempty"`
	RecommendedAction *integHlthRecommendedAction `json:"recommended_action,omitempty"`
}

type integHlthRecommendedAction struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func newOverviewCmd() *overviewCmd {
	oc := &overviewCmd{}

	oc.cmd = &cobra.Command{
		Use:   "overview",
		Args:  validators.NoArgs,
		Short: "Show integration health alerts for your account",
		Long: `Show integration health alerts Stripe has detected for your account,
along with why each matters and what to do about it.

Uses the mode of your active account context. Alerts are only available in
live mode and sandboxes, so pass --live if your active context is test mode.`,
		Example: `stripe overview
  stripe overview --live
  stripe overview --format json`,
		RunE: oc.runOverviewCmd,
	}

	oc.cmd.Flags().StringVar(&oc.format, "format", "", "Output format: 'json' for the raw API response")
	oc.cmd.Flags().IntVar(&oc.limit, "limit", 0, "Maximum number of alerts to return")
	oc.cmd.Flags().BoolVar(&oc.livemode, "live", false, "Show live mode alerts (default: your active context's mode). Requires that you're logged in to live mode — check with 'stripe whoami', switch with 'stripe switch'")
	oc.cmd.Flags().StringVar(&oc.stripeAccount, "stripe-account", "", "Connected account (acct_...) to show alerts for")
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

	var creds stripe.Credentials
	var err error
	if oc.livemode {
		creds, err = Config.GetProfile().ResolveCredentials(true)
	} else {
		creds, err = Config.GetProfile().ResolveCredentialsForAnyMode(false)
	}
	if err != nil {
		return err
	}

	resp, err := fetchIntegHlthAlerts(cmd.Context(), oc.apiBaseURL, creds, oc.stripeAccount, oc.limit)
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

func fetchIntegHlthAlerts(ctx context.Context, apiBaseURL string, creds stripe.Credentials, stripeAccount string, limit int) (*integHlthAlertsResponse, error) {
	if err := stripe.ValidateAPIBaseURL(apiBaseURL); err != nil {
		return nil, err
	}

	baseURL, err := url.Parse(apiBaseURL)
	if err != nil {
		return nil, err
	}

	client := &stripe.Client{
		BaseURL:     baseURL,
		Credentials: creds,
	}

	query := url.Values{}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}

	resp, err := client.PerformRequest(ctx, http.MethodGet, integHlthAlertsPath, query.Encode(), func(req *http.Request) error {
		req.Header.Set("Stripe-Version", integHlthAPIVersion)
		creds.ApplyAccountContextHeaders(req.Header, stripeAccount, "")
		return nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, errorcategory.Errorf(errorcategory.Auth, "credentials rejected (status 401); run `stripe login`")
	case resp.StatusCode != http.StatusOK:
		return nil, integHlthAPIError(resp.StatusCode, body)
	}

	var parsed integHlthAlertsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errorcategory.Errorf(errorcategory.API, "failed to parse response: %w", err)
	}
	return &parsed, nil
}

func integHlthAPIError(statusCode int, body []byte) error {
	var apiErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &apiErr) != nil || apiErr.Error.Message == "" {
		return errorcategory.Errorf(errorcategory.API, "failed to retrieve integration health alerts (status %d): %s", statusCode, strings.TrimSpace(string(body)))
	}
	if apiErr.Error.Code == v2TestmodeUnsupportedCode {
		return errorcategory.UserInputErrorf("integration health alerts aren't available in test mode; re-run with --live, or run `stripe switch` to select a sandbox")
	}
	return errorcategory.Errorf(errorcategory.API, "failed to retrieve integration health alerts (status %d): %s", statusCode, apiErr.Error.Message)
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
