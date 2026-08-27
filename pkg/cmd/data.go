package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const dataMetricsRunPath = "/v2/data/analytics/metric_query"

type dataCmd struct {
	cmd *cobra.Command
}

type dataMetricsCmd struct {
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
		Short: "Access Stripe Data APIs (Private Preview)",
		Long: `Access Stripe Data APIs.

Use the metrics subcommands to query time-series Stripe metric data. This
namespace is a Private Preview API.`,
		Args: validators.NoArgs,
	}
	dc.cmd.SetUsageTemplate(previewUsageTemplate())

	dc.cmd.AddCommand(newDataMetricsCmd().cmd)
	return dc
}

func newDataMetricsCmd() *dataMetricsCmd {
	mc := &dataMetricsCmd{}
	mc.cmd = &cobra.Command{
		Use:   "metrics",
		Short: "Query Stripe metrics (Private Preview)",
		Long: `Query time-series Stripe metric data.

Use the run subcommand to execute a metric query. This uses the
/v2/data/analytics/metric_query Private Preview API — the Stripe-Version
preview header is set automatically.

Metrics are specified by namespace.metric (e.g. revenue.mrr, revenue.arr).
See the supported metrics at https://docs.stripe.com/data/analytics/supported-metrics
and the API reference at
https://docs.stripe.com/api/v2/data/analytics/metric-query-results/create?api-version=preview`,
		Args: validators.NoArgs,
	}
	mc.cmd.AddCommand(newDataMetricsRunCmd().cmd)
	return mc
}

// previewUsageTemplate is the help template for the hand-written preview
// command trees (data, reporting query-runs).
//
// Usage and the trailing hint use HasSubCommands so parent --help stays
// populated if a child is later hidden. Available commands lists only
// non-hidden children via IsAvailableCommand.
//
// Set this explicitly on the root of each tree. Cobra otherwise inherits a
// usage template from the nearest ancestor that sets one, and the templates
// differ in whether they indent {{.Example}} — so a tree that relies on
// inheritance silently changes indentation if it is ever reparented.
func previewUsageTemplate() string {
	return fmt.Sprintf(`%s{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

%s
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s
  {{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding}} {{.Short}}{{end}}{{end}}{{end}}{{AIAgentHelp .}}{{if .HasAvailableLocalFlags}}

%s
{{WrappedLocalFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s
{{WrappedInheritedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`,
		ansi.Bold("Usage:"),
		ansi.Bold("Aliases:"),
		ansi.Bold("Examples:"),
		ansi.Bold("Available commands:"),
		ansi.Bold("Flags:"),
		ansi.Bold("Global flags:"),
	)
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
		Short: "Run a Stripe metric query (Private Preview)",
		Long: `Run a query for time-series Stripe metric data.

Sends a POST request to /v2/data/analytics/metric_query. This is a
Private Preview API — the Stripe-Version preview header is set automatically.

Metrics are specified by namespace.metric (e.g. revenue.mrr, revenue.arr).
Multiple metrics can be queried together as long as they share the same
namespace. Use --group-by to break down results by a dimension (at most one)
and --filter to restrict results to specific dimension values. Which
dimensions a metric supports varies by metric, and an unsupported one is
rejected, so check the metric's documentation before relying on it.

Required: --metric, --starts-at, --ends-at. --granularity defaults to day.
Optional: --currency, --timezone, --group-by, --filter, --limit. Only --metric
is checked locally; the API validates everything else.

Reading the results:
  * A bucket's timestamp marks the END of its period, not the start. January
    2026 at --granularity month is timestamped 2026-01-31, and the week of
    March 1-7 is timestamped 2026-03-07. A period still in progress is
    timestamped with the most recent day that has data.
  * Buckets align to the result timezone (your account's, or --timezone), so
    the first bucket can look like it precedes --starts-at. For an account in
    UTC-8, --starts-at 2026-03-01T00:00:00Z is 16:00 on 2026-02-28 locally, so
    the first daily bucket is that local day. Pass --timezone UTC to align
    buckets to the UTC timestamps you asked for.
  * API errors name dimensions with an _id suffix: --filter "price=..." is
    reported against price_id.

See the supported metrics at https://docs.stripe.com/data/analytics/supported-metrics
and the API reference at
https://docs.stripe.com/api/v2/data/analytics/metric-query-results/create?api-version=preview`,
		Example: `# Query daily MRR for March 2026
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

  # Break MRR down by a dimension the metric supports
  stripe data metrics run \
    --metric revenue.mrr \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-01-31T23:59:59Z \
    --granularity month \
    --currency usd \
    --group-by subscription

  # Filter by price — replace price_<id> with a Price id from your account
  stripe data metrics run \
    --metric usage_based_billing.gross_usage_revenue \
    --starts-at 2026-01-01T00:00:00Z \
    --ends-at 2026-06-30T23:59:59Z \
    --granularity month \
    --filter "price=price_<id>"`,
		RunE: c.runDataMetricsRunCmd,
	}

	c.cmd.Flags().StringArrayVar(&c.metrics, "metric", []string{}, "Metric to query: namespace.metric name (e.g. revenue.mrr) or ID (e.g. metric_<id>). Repeatable.")
	c.cmd.Flags().StringVar(&c.startsAt, "starts-at", "", "Start of the time range as an ISO 8601 datetime (e.g. 2026-01-01T00:00:00Z)")
	c.cmd.Flags().StringVar(&c.endsAt, "ends-at", "", "End of the time range as an ISO 8601 datetime (e.g. 2026-01-31T23:59:59Z)")
	c.cmd.Flags().StringVar(&c.granularity, "granularity", "day", "Time granularity: day, week, month, or year")
	c.cmd.Flags().StringArrayVar(&c.groupBy, "group-by", []string{}, "Dimension to group by (at most one). Which dimensions a metric supports varies by metric; an unsupported one is rejected. See https://docs.stripe.com/data/analytics/supported-metrics")
	c.cmd.Flags().StringArrayVar(&c.filters, "filter", []string{}, "Filter results by dimension values, in key=value format (repeatable). E.g. --filter \"price=price_<id>\"")
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
		return errorcategory.Errorf(errorcategory.UserInput, "at least one --metric is required")
	}

	// A dry run sends nothing, so resolve credentials leniently: the live/sandbox
	// context gate must not be the thing that stops you inspecting a request.
	resolve := c.rb.ResolveCredentials
	if c.rb.DryRun {
		resolve = c.rb.ResolveCredentialsForPreview
	}

	creds, err := resolve()
	if err != nil {
		return err
	}

	// Forward the remaining parameters to the API and let it validate them.
	body, err := c.buildRequestBody(cmd.Flags().Changed("limit"))
	if err != nil {
		return err
	}

	if c.rb.DryRun {
		output, err := c.rb.BuildDryRunOutput(creds, c.rb.APIBaseURL, dataMetricsRunPath, &requests.RequestParameters{}, body)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	_, err = c.rb.MakeRequest(cmd.Context(), creds, dataMetricsRunPath, &requests.RequestParameters{}, body, true, nil)
	return formatMetricQueryError(err)
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
			return nil, errorcategory.Errorf(errorcategory.UserInput, "invalid filter %q: must be in key=value format (e.g. --filter \"currency=usd\")", f)
		}
		result[parts[0]] = append(result[parts[0]], parts[1])
	}
	return result, nil
}
