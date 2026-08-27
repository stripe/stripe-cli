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

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const queryRunsPath = "/v2/data/reporting/query_runs"

// reportingNamespaceShort and reportingNamespaceLong describe the `reporting`
// namespace, which mixes the generated Sigma API resources (report_runs,
// report_types) with the hand-written query-runs commands.
const reportingNamespaceShort = "Run Sigma report and ad hoc SQL queries"

const reportingNamespaceLong = `Access Stripe reporting APIs.

Use the query-runs subcommands to run ad hoc SQL queries against your Stripe
data via the /v2/data/reporting/query_runs Public Preview API. Use the
report_runs and report_types resources for scheduled Sigma reports.`

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

// addReportingQueryRunsCmd hangs query-runs off the generated `reporting`
// API-resource namespace rather than registering a second top-level command
// with the same name.
//
// Two same-named root children make cobra's Find resolve `stripe reporting
// report_runs list` to whichever sibling it reaches first. Because that
// sibling has no `report_runs` child and no Run function, cobra printed the
// wrong help and exited 0 — so the Sigma commands were silently unreachable
// (and Args validation never ran, since Runnable() is checked first).
//
// Call this after the generated resource commands are registered.
func addReportingQueryRunsCmd(rootCmd *cobra.Command) {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() != "reporting" {
			continue
		}
		// The generated namespace carries no descriptions; supply them so
		// `--help` and `--map` agree on what the namespace does.
		if cmd.Short == "" {
			cmd.Short = reportingNamespaceShort
		}
		if cmd.Long == "" {
			cmd.Long = reportingNamespaceLong
		}
		cmd.AddCommand(newReportingQueryRunsCmd().cmd)
		return
	}

	// No generated namespace to merge into (e.g. a trimmed resource set):
	// register a top-level command so query-runs stays reachable.
	rootCmd.AddCommand(newReportingCmd().cmd)
}

func newReportingCmd() *reportingCmd {
	rc := &reportingCmd{}
	rc.cmd = &cobra.Command{
		Use:   "reporting",
		Short: reportingNamespaceShort,
		Long:  reportingNamespaceLong,
		Args:  validators.NoArgs,
	}

	rc.cmd.AddCommand(newReportingQueryRunsCmd().cmd)
	return rc
}

func newReportingQueryRunsCmd() *reportingQueryRunsCmd {
	qrc := &reportingQueryRunsCmd{}
	qrc.cmd = &cobra.Command{
		Use:   "query-runs",
		Short: "Create and retrieve QueryRun objects (Public Preview)",
		Long: `Create and retrieve QueryRun objects.

A QueryRun runs a custom SQL query against your Stripe data. Create a query run
to kick off a query, then retrieve it to poll its status and fetch the download
URL of the result once the query has completed.`,
		Args: validators.NoArgs,
	}
	// Set explicitly: this subtree hangs off the generated `reporting`
	// namespace, whose template renders {{.Example}} unindented.
	qrc.cmd.SetUsageTemplate(previewUsageTemplate())

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
		Short: "Create a query run from custom SQL (Public Preview)",
		Long: `Create a query run to execute a custom, ad hoc SQL query against your Stripe data.

Sends a POST request to /v2/data/reporting/query_runs. This is a Public Preview
API — the Stripe-Version preview header is set automatically.

The query runs asynchronously. The response contains the query run's id and
status ("running", "succeeded", or "failed"); poll it with
"stripe reporting query-runs retrieve <id>" until the status is "succeeded".
The output is then at result.file.download_url.url — download_url is an object
holding the url and its expires_at, so the url itself is one level deeper than
the field name suggests. The link is short-lived (a few minutes); retrieve the
query run again for a fresh one.

Provide the SQL inline with --sql, from a file with --sql-file, or via stdin
by passing --sql-file -.

Nested API fields use bracket notation as flags (for example
--result_options[compress_file]=true), not dotted names from the API
reference (--result_options.compress_file). Prefer the dedicated
--compress-file flag when one exists.`,
		Example: `# Run an ad hoc query
  stripe reporting query-runs create --sql "SELECT * FROM charges LIMIT 10"

  # Compress the result file
  stripe reporting query-runs create --sql "SELECT * FROM charges LIMIT 10" --compress-file`,
		RunE: cc.runReportingQueryRunsCreateCmd,
		Args: validators.NoArgs,
	}

	cc.cmd.Flags().StringVar(&cc.sql, "sql", "", "The SQL query to run [required unless --sql-file is set]")
	cc.cmd.Flags().StringVar(&cc.sqlFile, "sql-file", "", "Path to a file containing the SQL query to run. Use \"-\" to read from stdin.")
	cc.cmd.Flags().BoolVar(&cc.compressFile, "compress-file", false, "Compress the result file. Equivalent to the API parameter result_options.compress_file; the raw form is --result_options[compress_file]=true.")
	// Hidden alias so the bracket form from the API docs is accepted (pflag
	// treats --result_options[compress_file] as a flag name).
	cc.cmd.Flags().BoolVar(&cc.compressFile, "result_options[compress_file]", false, "")
	cc.cmd.Flags().MarkHidden("result_options[compress_file]") // #nosec G104

	cc.cmd.SetFlagErrorFunc(flagErrorWithNestedAPIHint)

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
		Short: "Retrieve a query run (Public Preview)",
		Long: `Retrieve a query run by its id to check its status and fetch results.

Sends a GET request to /v2/data/reporting/query_runs/{id}. This is a
Public Preview API — the Stripe-Version preview header is set automatically.

Once the query run's status is "succeeded", the output can be downloaded from
result.file.download_url.url. download_url is an object holding the url and its
expires_at, so the url itself is one level deeper than the field name suggests.
The link is short-lived (a few minutes); retrieve the query run again for a
fresh one.`,
		Example: `# Retrieve a query run (replace <query_run_id> with an id from create)
  stripe reporting query-runs retrieve <query_run_id>`,
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

	// A dry run sends nothing, so resolve credentials leniently: the live/sandbox
	// context gate must not be the thing that stops you inspecting a request.
	resolve := cc.rb.ResolveCredentials
	if cc.rb.DryRun {
		resolve = cc.rb.ResolveCredentialsForPreview
	}

	creds, err := resolve()
	if err != nil {
		return err
	}

	body := cc.buildRequestBody(sql)

	if cc.rb.DryRun {
		output, err := cc.rb.BuildDryRunOutput(creds, cc.rb.APIBaseURL, queryRunsPath, &requests.RequestParameters{}, body)
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

	_, err = cc.rb.MakeRequest(cmd.Context(), creds, queryRunsPath, &requests.RequestParameters{}, body, true, nil)
	return err
}

// resolveSQL determines the SQL query to run from the --sql / --sql-file flags.
// Exactly one source must be provided.
func (cc *reportingQueryRunsCreateCmd) resolveSQL(cmd *cobra.Command) (string, error) {
	if cc.sql != "" && cc.sqlFile != "" {
		return "", errorcategory.Errorf(errorcategory.UserInput, "--sql and --sql-file are mutually exclusive")
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
			return "", errorcategory.Errorf(errorcategory.UserInput, "no SQL found in %q", cc.sqlFile)
		}
		return sql, nil
	}

	return "", errorcategory.Errorf(errorcategory.UserInput, "one of --sql or --sql-file is required")
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

	creds, err := rc.rb.ResolveCredentials()
	if err != nil {
		return err
	}

	path := queryRunsPath + "/" + url.PathEscape(args[0])

	_, err = rc.rb.MakeRequest(cmd.Context(), creds, path, &requests.RequestParameters{}, nil, true, nil)
	return err
}

// readAllInput reads all data from the command's input stream (stdin).
func readAllInput(cmd *cobra.Command) ([]byte, error) {
	return io.ReadAll(cmd.InOrStdin())
}
