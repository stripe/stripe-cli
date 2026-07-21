package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const queryRunsPath = "/v2/data/reporting/query_runs"

type reportingCmd struct {
	cmd *cobra.Command
}

type reportingQueryRunsCmd struct {
	cmd *cobra.Command
}

type reportingQueryRunsCreateCmd struct {
	cmd          *cobra.Command
	rb           requests.Base
	sql          string
	sqlFile      string
	compressFile bool
}

type reportingQueryRunsRetrieveCmd struct {
	cmd *cobra.Command
	rb  requests.Base
}

func newReportingCmd() *reportingCmd {
	rc := &reportingCmd{}
	rc.cmd = &cobra.Command{
		Use:   "reporting",
		Short: "Run Stripe Sigma queries via the Data Reporting API",
		Long: `Run ad hoc SQL queries against your Stripe data using the Data Reporting API.

Use the query-runs subcommands to kick off a new query and retrieve its
results. This uses the /v2/data/reporting/query_runs preview API.`,
		Args: validators.NoArgs,
	}

	rc.cmd.AddCommand(newReportingQueryRunsCmd().cmd)
	return rc
}

func newReportingQueryRunsCmd() *reportingQueryRunsCmd {
	qrc := &reportingQueryRunsCmd{}
	qrc.cmd = &cobra.Command{
		Use:   "query-runs",
		Short: "Create and retrieve QueryRun objects",
		Long: `Create and retrieve QueryRun objects.

A QueryRun runs a custom SQL query against your Stripe data. Create a query run
to kick off a query, then retrieve it to poll its status and fetch the download
URL of the result once the query has completed.`,
		Args: validators.NoArgs,
	}

	qrc.cmd.AddCommand(newReportingQueryRunsCreateCmd().cmd)
	qrc.cmd.AddCommand(newReportingQueryRunsRetrieveCmd().cmd)
	return qrc
}

func newReportingQueryRunsCreateCmd() *reportingQueryRunsCreateCmd {
	cc := &reportingQueryRunsCreateCmd{}

	cc.rb = requests.Base{
		Method:           http.MethodPost,
		Profile:          &Config.Profile,
		IsPreviewCommand: true,
	}

	cc.cmd = &cobra.Command{
		Use:   "create",
		Short: "Create a query run from custom SQL",
		Long: `Create a query run to execute a custom, ad hoc SQL query against your Stripe data.

Sends a POST request to /v2/data/reporting/query_runs. This is a preview API —
the Stripe-Version preview header is set automatically.

The query runs asynchronously. The response contains the query run's id and
status ("running", "succeeded", or "failed"); poll it with
"stripe reporting query-runs retrieve <id>" until the status is "succeeded",
then use the result's download_url to fetch the output.

Provide the SQL inline with --sql, from a file with --sql-file, or via stdin
by passing --sql-file -.`,
		Example: `  # Run an ad hoc query
  stripe reporting query-runs create --sql "SELECT * FROM charges LIMIT 10"

  # Read the SQL from a file
  stripe reporting query-runs create --sql-file ./query.sql

  # Read the SQL from stdin
  cat query.sql | stripe reporting query-runs create --sql-file -`,
		RunE: cc.runReportingQueryRunsCreateCmd,
		Args: validators.NoArgs,
	}

	cc.cmd.Flags().StringVar(&cc.sql, "sql", "", "The SQL query to run [required unless --sql-file is set]")
	cc.cmd.Flags().StringVar(&cc.sqlFile, "sql-file", "", "Path to a file containing the SQL query to run. Use \"-\" to read from stdin.")
	cc.cmd.Flags().BoolVar(&cc.compressFile, "compress-file", false, "Compress the result file (sets result_options.compress_file)")

	cc.cmd.Flags().BoolVar(&cc.rb.DryRun, "dry-run", false, "Preview the request without sending it")
	cc.cmd.Flags().BoolVarP(&cc.rb.Livemode, "live", "", false, "Make a live request (default: test)")
	cc.cmd.Flags().BoolVarP(&cc.rb.DarkStyle, "dark-style", "", false, "Use a darker color scheme better suited for lighter command-lines")

	cc.cmd.Flags().StringVar(&cc.rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	cc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return cc
}

func newReportingQueryRunsRetrieveCmd() *reportingQueryRunsRetrieveCmd {
	rc := &reportingQueryRunsRetrieveCmd{}

	rc.rb = requests.Base{
		Method:           http.MethodGet,
		Profile:          &Config.Profile,
		IsPreviewCommand: true,
	}

	rc.cmd = &cobra.Command{
		Use:   "retrieve <id>",
		Short: "Retrieve a query run",
		Long: `Retrieve a query run by its id to check its status and fetch results.

Sends a GET request to /v2/data/reporting/query_runs/{id}. This is a preview
API — the Stripe-Version preview header is set automatically.

Once the query run's status is "succeeded", the result's download_url can be
used to download the query output.`,
		Example: `  # Retrieve a query run
  stripe reporting query-runs retrieve qryrun_test_123`,
		Args: validators.ExactArgs(1),
		RunE: rc.runReportingQueryRunsRetrieveCmd,
	}

	rc.cmd.Flags().BoolVarP(&rc.rb.Livemode, "live", "", false, "Make a live request (default: test)")
	rc.cmd.Flags().BoolVarP(&rc.rb.DarkStyle, "dark-style", "", false, "Use a darker color scheme better suited for lighter command-lines")

	rc.cmd.Flags().StringVar(&rc.rb.APIBaseURL, "api-base", stripe.DefaultAPIBaseURL, "Sets the API base URL")
	rc.cmd.Flags().MarkHidden("api-base") // #nosec G104

	return rc
}

func (cc *reportingQueryRunsCreateCmd) runReportingQueryRunsCreateCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(cc.rb.APIBaseURL); err != nil {
		return err
	}

	sql, err := cc.resolveSQL(cmd)
	if err != nil {
		return err
	}

	apiKey, err := cc.rb.Profile.GetAPIKey(cc.rb.Livemode)
	if err != nil {
		return err
	}

	body := cc.buildRequestBody(sql)

	if cc.rb.DryRun {
		output, err := cc.rb.BuildDryRunOutput(apiKey, cc.rb.APIBaseURL, queryRunsPath, &requests.RequestParameters{}, body)
		if err != nil {
			return err
		}
		b, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	}

	_, err = cc.rb.MakeRequest(cmd.Context(), apiKey, queryRunsPath, &requests.RequestParameters{}, body, true, nil)
	return err
}

// resolveSQL determines the SQL query to run from the --sql / --sql-file flags.
// Exactly one source must be provided.
func (cc *reportingQueryRunsCreateCmd) resolveSQL(cmd *cobra.Command) (string, error) {
	if cc.sql != "" && cc.sqlFile != "" {
		return "", fmt.Errorf("--sql and --sql-file are mutually exclusive")
	}

	if cc.sql != "" {
		return cc.sql, nil
	}

	if cc.sqlFile != "" {
		var raw []byte
		var err error
		if cc.sqlFile == "-" {
			raw, err = readAllInput(cmd)
		} else {
			raw, err = os.ReadFile(cc.sqlFile)
		}
		if err != nil {
			return "", fmt.Errorf("failed to read SQL from %q: %w", cc.sqlFile, err)
		}
		sql := strings.TrimSpace(string(raw))
		if sql == "" {
			return "", fmt.Errorf("no SQL found in %q", cc.sqlFile)
		}
		return sql, nil
	}

	return "", fmt.Errorf("one of --sql or --sql-file is required")
}

func (cc *reportingQueryRunsCreateCmd) buildRequestBody(sql string) map[string]interface{} {
	body := map[string]interface{}{
		"sql": sql,
	}

	if cc.compressFile {
		body["result_options"] = map[string]interface{}{
			"compress_file": true,
		}
	}

	return body
}

func (rc *reportingQueryRunsRetrieveCmd) runReportingQueryRunsRetrieveCmd(cmd *cobra.Command, args []string) error {
	if err := stripe.ValidateAPIBaseURL(rc.rb.APIBaseURL); err != nil {
		return err
	}

	apiKey, err := rc.rb.Profile.GetAPIKey(rc.rb.Livemode)
	if err != nil {
		return err
	}

	path := queryRunsPath + "/" + url.PathEscape(args[0])

	_, err = rc.rb.MakeRequest(cmd.Context(), apiKey, path, &requests.RequestParameters{}, nil, true, nil)
	return err
}

// readAllInput reads all data from the command's input stream (stdin).
func readAllInput(cmd *cobra.Command) ([]byte, error) {
	return io.ReadAll(cmd.InOrStdin())
}
