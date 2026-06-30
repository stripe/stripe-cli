package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const analyticsQueryPath = "/v2/data/analytics/metric_query"

type analyticsCmd struct {
	cmd *cobra.Command
}

type analyticsQueryCmd struct {
	cmd         *cobra.Command
	rb          requests.Base
	metrics     []string
	startsAt    string
	endsAt      string
	granularity string
	groupBy     []string
	filters     []string
	currency    string
	timezone    string
	limit       int
}

func newAnalyticsCmd() *analyticsCmd {
	ac := &analyticsCmd{}
	ac.cmd = &cobra.Command{
		Use:   "analytics",
		Short: "Query Stripe analytics metrics",
		Long: `Query Stripe analytics metrics using the Analytics API.

Use subcommands to query time-series metric data for your Stripe account.`,
		Args: validators.NoArgs,
	}

	ac.cmd.AddCommand(newAnalyticsQueryCmd().cmd)
	return ac
}

func newAnalyticsQueryCmd() *analyticsQueryCmd {
	aqc := &analyticsQueryCmd{}

	aqc.rb = requests.Base{
		Method:           http.MethodPost,
		Profile:          &Config.Profile,
		IsPreviewCommand: true,
	}

	aqc.cmd = &cobra.Command{
		Use:   "query",
		Short: "Query a Stripe analytics metric",
		Long: `Query time-series data for a Stripe analytics metric.

Sends a POST request to /v2/data/analytics/metric_query. This is a preview
API — the Stripe-Version preview header is set automatically.

Metrics are specified by name (e.g. revenue.mrr, revenue.arr). Multiple metrics
can be queried together as long as they share the same namespace (e.g. revenue.mrr
and revenue.arr both belong to the "revenue" namespace). Use --group-by to break
down results by a dimension (e.g. price, product, customer). Use --filter to
restrict results to specific dimension values.

See https://docs.stripe.com/data/analytics for supported metrics.`,
		Example: `  # Query daily MRR for March 2026
  stripe analytics query \
    --metric revenue.mrr \
    --starts-at 2026-03-01T00:00:00Z \
    --ends-at 2026-03-31T23:59:59Z \
    --granularity day

  # Query MRR and ARR together (same namespace)
  stripe analytics query \
    --metric revenue.mrr \
    --metric revenue.arr \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-01-31T23:59:59Z \
    --granularity month \
    --currency usd

  # Group by product dimension
  stripe analytics query \
    --metric revenue.mrr \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-01-31T23:59:59Z \
    --granularity month \
    --currency usd \
    --group-by product

  # Filter results to specific prices
  stripe analytics query \
    --metric usage_based_billing.gross_usage_revenue \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-06-30T23:59:59Z \
    --granularity month \
    --filter "price=price_HG4jxINMyHorkI" \
    --filter "price=price_HG4UtyJbuJIsVt"`,
		RunE: aqc.runAnalyticsQueryCmd,
	}

	aqc.cmd.Flags().StringArrayVar(&aqc.metrics, "metric", []string{}, "Metric to query: common name (e.g. revenue.mrr) or ID (e.g. metric_61Sud3n5oAGVCWiSr5). Repeatable; all metrics must share the same namespace. [required]")
	aqc.cmd.Flags().StringVar(&aqc.startsAt, "starts-at", "", "Start of the time range in RFC3339 format (e.g. 2026-01-01T00:00:00Z) [required]")
	aqc.cmd.Flags().StringVar(&aqc.endsAt, "ends-at", "", "End of the time range in RFC3339 format (e.g. 2026-01-31T23:59:59Z) [required]")
	aqc.cmd.Flags().StringVar(&aqc.granularity, "granularity", "day", "Time granularity: day, week, month, or year")
	aqc.cmd.Flags().StringArrayVar(&aqc.groupBy, "group-by", []string{}, "Dimension to group results by (e.g. price, product, customer). At most one allowed.")
	aqc.cmd.Flags().StringArrayVar(&aqc.filters, "filter", []string{}, "Filter results by dimension values, in key=value format (repeatable). E.g. --filter \"price=price_abc123\"")
	aqc.cmd.Flags().StringVar(&aqc.currency, "currency", "", "Currency code to convert monetary values to (e.g. usd, eur)")
	aqc.cmd.Flags().StringVar(&aqc.timezone, "timezone", "", "Timezone for result alignment (e.g. America/New_York). Defaults to your account timezone.")
	aqc.cmd.Flags().IntVar(&aqc.limit, "limit", 0, "Maximum number of rows to return (1–1000). Default is all rows.")

	aqc.cmd.Flags().BoolVar(&aqc.rb.DryRun, "dry-run", false, "Preview the request without sending it")
	aqc.cmd.Flags().BoolVarP(&aqc.rb.Livemode, "live", "", false, "Make a live request (default: test)")
	aqc.cmd.Flags().BoolVarP(&aqc.rb.DarkStyle, "dark-style", "", false, "Use a darker color scheme better suited for lighter command-lines")

	aqc.cmd.Flags().StringVar(&aqc.rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	aqc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return aqc
}

func (aqc *analyticsQueryCmd) runAnalyticsQueryCmd(cmd *cobra.Command, args []string) error {
	if len(aqc.metrics) == 0 {
		return fmt.Errorf("--metric is required")
	}
	if aqc.startsAt == "" {
		return fmt.Errorf("--starts-at is required")
	}
	if aqc.endsAt == "" {
		return fmt.Errorf("--ends-at is required")
	}
	if len(aqc.groupBy) > 1 {
		return fmt.Errorf("--group-by accepts at most one dimension, got %d: %s", len(aqc.groupBy), strings.Join(aqc.groupBy, ", "))
	}

	apiKey, err := aqc.rb.Profile.GetAPIKey(aqc.rb.Livemode)
	if err != nil {
		return err
	}

	body, err := aqc.buildRequestBody()
	if err != nil {
		return err
	}

	if aqc.rb.DryRun {
		output, err := aqc.rb.BuildDryRunOutput(apiKey, aqc.rb.APIBaseURL, analyticsQueryPath, &requests.RequestParameters{}, body)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	_, err = aqc.rb.MakeRequest(cmd.Context(), apiKey, analyticsQueryPath, &requests.RequestParameters{}, body, true, nil)
	return err
}

func (aqc *analyticsQueryCmd) buildRequestBody() (map[string]interface{}, error) {
	metricObjs := make([]map[string]interface{}, len(aqc.metrics))
	for i, m := range aqc.metrics {
		metricObjs[i] = metricRef(m)
	}

	body := map[string]interface{}{
		"metrics":     metricObjs,
		"starts_at":   aqc.startsAt,
		"ends_at":     aqc.endsAt,
		"granularity": aqc.granularity,
	}

	if aqc.currency != "" {
		body["currency"] = aqc.currency
	}

	if aqc.timezone != "" {
		body["timezone"] = aqc.timezone
	}

	if aqc.limit > 0 {
		body["limit"] = aqc.limit
	}

	if len(aqc.groupBy) > 0 {
		body["group_by"] = aqc.groupBy
	}

	if len(aqc.filters) > 0 {
		parsed, err := parseAnalyticsFilters(aqc.filters)
		if err != nil {
			return nil, err
		}
		body["filters"] = parsed
	}

	return body, nil
}

// metricRef builds the metric object for the API request body.
// A value is treated as an ID when it starts with "metric_" and contains no dot;
// common names always contain a dot (namespace.metric_name) so a value like
// "metric_x.foo" is correctly sent as {"name": ...} rather than {"id": ...}.
func metricRef(value string) map[string]interface{} {
	if strings.HasPrefix(value, "metric_") && !strings.Contains(value, ".") {
		return map[string]interface{}{"id": value}
	}
	return map[string]interface{}{"name": value}
}

// parseAnalyticsFilters parses --filter "key=value" flags into the map[string][]string
// shape expected by the analytics API (e.g. {"price": ["price_abc", "price_xyz"]}).
func parseAnalyticsFilters(filters []string) (map[string][]string, error) {
	result := make(map[string][]string)
	for _, f := range filters {
		parts := strings.SplitN(f, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid filter %q: must be in key=value format (e.g. --filter \"currency=usd\")", f)
		}
		result[parts[0]] = append(result[parts[0]], parts[1])
	}
	return result, nil
}
