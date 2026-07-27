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

const dataMetricsRunPath = "/v2/data/analytics/metric_query"

type dataCmd struct {
	cmd *cobra.Command
}

type dataMetricsRunCmd struct {
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

func newDataCmd() *dataCmd {
	dc := &dataCmd{}
	dc.cmd = &cobra.Command{
		Use:   "data",
		Short: "Access Stripe Data APIs",
		// Private Preview API — hidden until GA.
		Hidden: true,
		Args:   validators.NoArgs,
	}

	metricsCmd := &cobra.Command{
		Use:   "metrics",
		Short: "Query Stripe metrics",
		// Private Preview API — hidden until GA.
		Hidden: true,
		Args:   validators.NoArgs,
	}
	metricsCmd.AddCommand(newDataMetricsRunCmd().cmd)

	dc.cmd.AddCommand(metricsCmd)
	return dc
}

func newDataMetricsRunCmd() *dataMetricsRunCmd {
	c := &dataMetricsRunCmd{}

	c.rb = requests.Base{
		Method:           http.MethodPost,
		Profile:          &Config.Profile,
		IsPreviewCommand: true,
	}

	c.cmd = &cobra.Command{
		Use:   "run",
		Short: "Run a Stripe metric query",
		// Private Preview API — hidden until GA.
		Hidden: true,
		Long: `Run a query for time-series Stripe metric data.

Sends a POST request to /v2/data/analytics/metric_query. This is a Private
Preview API — the Stripe-Version preview header is set automatically.

Metrics are specified by namespace.metric (e.g. revenue.mrr, revenue.arr).
Multiple metrics can be queried together as long as they share the same
namespace. Use --group-by to break down results by a dimension (e.g. price,
product, customer). Use --filter to restrict results to specific dimension
values.

Required API fields: metrics, starts_at, ends_at, granularity. Optional:
currency, timezone, group_by, filters, limit. The API validates all parameters.

See the supported metrics at https://docs.stripe.com/data/analytics/supported-metrics
and the API reference at
https://docs.stripe.com/api/v2/data/analytics/metric-query-results/create?api-version=preview`,
		Example: `  # Query daily MRR for March 2026
  stripe data metrics run \
    --metric revenue.mrr \
    --starts-at 2026-03-01T00:00:00Z \
    --ends-at 2026-03-31T23:59:59Z \
    --granularity day

  # Query MRR and ARR together (same namespace)
  stripe data metrics run \
    --metric revenue.mrr \
    --metric revenue.arr \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-01-31T23:59:59Z \
    --granularity month \
    --currency usd

  # Group by product dimension
  stripe data metrics run \
    --metric revenue.mrr \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-01-31T23:59:59Z \
    --granularity month \
    --currency usd \
    --group-by product

  # Filter results to specific prices
  stripe data metrics run \
    --metric usage_based_billing.gross_usage_revenue \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-06-30T23:59:59Z \
    --granularity month \
    --filter "price=price_abc123" \
    --filter "price=price_xyz789"`,
		RunE: c.runDataMetricsRunCmd,
	}

	c.cmd.Flags().StringArrayVar(&c.metrics, "metric", []string{}, "Metric to query: namespace.metric name (e.g. revenue.mrr) or ID (e.g. metric_61Sud3n5oAGVCWiSr5). Repeatable.")
	c.cmd.Flags().StringVar(&c.startsAt, "starts-at", "", "Start of the time range as an ISO 8601 datetime (e.g. 2026-01-01T00:00:00Z)")
	c.cmd.Flags().StringVar(&c.endsAt, "ends-at", "", "End of the time range as an ISO 8601 datetime (e.g. 2026-01-31T23:59:59Z)")
	c.cmd.Flags().StringVar(&c.granularity, "granularity", "day", "Time granularity: day, week, month, or year")
	c.cmd.Flags().StringArrayVar(&c.groupBy, "group-by", []string{}, "Dimension to group results by (e.g. price, product, customer)")
	c.cmd.Flags().StringArrayVar(&c.filters, "filter", []string{}, "Filter results by dimension values, in key=value format (repeatable). E.g. --filter \"price=price_abc123\"")
	c.cmd.Flags().StringVar(&c.currency, "currency", "", "Currency code to convert monetary values to (e.g. usd, eur). Defaults to your account's default currency.")
	c.cmd.Flags().StringVar(&c.timezone, "timezone", "", "Timezone for result alignment (e.g. America/New_York). Defaults to your account timezone.")
	c.cmd.Flags().IntVar(&c.limit, "limit", 0, "Maximum number of rows to return (1–1000). Default is all rows.")

	c.cmd.Flags().BoolVar(&c.rb.DryRun, "dry-run", false, "Preview the request without sending it")
	c.cmd.Flags().BoolVarP(&c.rb.Livemode, "live", "", false, "Make a live request (default: test)")
	c.cmd.Flags().BoolVarP(&c.rb.DarkStyle, "dark-style", "", false, "Use a darker color scheme better suited for lighter command-lines")

	// --api-base overrides the API host (used for local/dev testing); it's hidden
	// from help. MarkHidden only errors on an unknown flag name, so the returned
	// error is intentionally ignored (#nosec G104 silences the gosec warning).
	c.cmd.Flags().StringVar(&c.rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	c.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return c
}

func (c *dataMetricsRunCmd) runDataMetricsRunCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(c.rb.APIBaseURL); err != nil {
		return err
	}

	// Guard only the primary input: with no --metric the API receives an empty
	// metrics array and returns an opaque error. All other rules (time ranges,
	// namespaces, group-by/filter cardinality, limit bounds, etc.) are left to
	// the API so we don't duplicate logic that could drift out of sync.
	if len(c.metrics) == 0 {
		return fmt.Errorf("at least one --metric is required")
	}

	apiKey, err := c.rb.Profile.GetAPIKey(c.rb.Livemode)
	if err != nil {
		return err
	}

	// Forward the remaining parameters to the API and let it validate them.
	body, err := c.buildRequestBody(cmd.Flags().Changed("limit"))
	if err != nil {
		return err
	}

	if c.rb.DryRun {
		output, err := c.rb.BuildDryRunOutput(apiKey, c.rb.APIBaseURL, dataMetricsRunPath, &requests.RequestParameters{}, body)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	_, err = c.rb.MakeRequest(cmd.Context(), apiKey, dataMetricsRunPath, &requests.RequestParameters{}, body, true, nil)
	return err
}

// buildRequestBody assembles the JSON body for the metric query request.
// includeLimit forwards the --limit value (whatever it is) only when the user
// explicitly set the flag, so the API can validate the bound instead of us
// silently dropping values here.
func (c *dataMetricsRunCmd) buildRequestBody(includeLimit bool) (map[string]interface{}, error) {
	metricObjs := make([]map[string]interface{}, len(c.metrics))
	for i, m := range c.metrics {
		metricObjs[i] = metricRef(m)
	}

	body := map[string]interface{}{
		"metrics":     metricObjs,
		"starts_at":   c.startsAt,
		"ends_at":     c.endsAt,
		"granularity": c.granularity,
	}

	if c.currency != "" {
		body["currency"] = c.currency
	}

	if c.timezone != "" {
		body["timezone"] = c.timezone
	}

	if includeLimit {
		body["limit"] = c.limit
	}

	if len(c.groupBy) > 0 {
		body["group_by"] = c.groupBy
	}

	if len(c.filters) > 0 {
		parsed, err := parseMetricFilters(c.filters)
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

// parseMetricFilters parses --filter "key=value" flags into the map[string][]string
// shape expected by the analytics API (e.g. {"price": ["price_abc", "price_xyz"]}).
func parseMetricFilters(filters []string) (map[string][]string, error) {
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
