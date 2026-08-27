package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

// --- Unit tests: buildRequestBody ---

func TestReportingBuildRequestBody_Minimal(t *testing.T) {
	cc := &reportingQueryRunsCreateCmd{}
	body := cc.buildRequestBody("SELECT * FROM charges LIMIT 10")

	assert.Equal(t, "SELECT * FROM charges LIMIT 10", body["sql"])
	assert.Nil(t, body["result_options"])
}

func TestReportingBuildRequestBody_CompressFile(t *testing.T) {
	cc := &reportingQueryRunsCreateCmd{compressFile: true}
	body := cc.buildRequestBody("SELECT 1")

	resultOptions := body["result_options"].(map[string]interface{})
	assert.Equal(t, true, resultOptions["compress_file"])
}

// --- Unit tests: resolveSQL ---

func TestResolveSQL_Inline(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	cc.sql = "SELECT 1"

	sql, err := cc.resolveSQL(cc.cmd)
	require.NoError(t, err)
	assert.Equal(t, "SELECT 1", sql)
}

func TestResolveSQL_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "query.sql")
	require.NoError(t, os.WriteFile(path, []byte("\n  SELECT * FROM charges  \n"), 0600))

	cc := newReportingQueryRunsCreateCmd()
	cc.sqlFile = path

	sql, err := cc.resolveSQL(cc.cmd)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM charges", sql, "file contents should be trimmed")
}

func TestResolveSQL_FromStdin(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	cc.sqlFile = "-"
	cc.cmd.SetIn(strings.NewReader("SELECT 42\n"))

	sql, err := cc.resolveSQL(cc.cmd)
	require.NoError(t, err)
	assert.Equal(t, "SELECT 42", sql)
}

func TestResolveSQL_MissingBoth(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()

	_, err := cc.resolveSQL(cc.cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "one of --sql or --sql-file is required")
}

func TestResolveSQL_MutuallyExclusive(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	cc.sql = "SELECT 1"
	cc.sqlFile = "query.sql"

	_, err := cc.resolveSQL(cc.cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestResolveSQL_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.sql")
	require.NoError(t, os.WriteFile(path, []byte("   \n"), 0600))

	cc := newReportingQueryRunsCreateCmd()
	cc.sqlFile = path

	_, err := cc.resolveSQL(cc.cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no SQL found")
}

// The CLI intentionally does not validate SQL client-side; syntactically
// invalid or non-SQL content is passed through verbatim (after trimming) and
// the API is responsible for rejecting it (query_run_invalid_sql). This test
// documents that pass-through behavior.
func TestResolveSQL_NonSQLContentPassedThrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notsql.txt")
	content := "this is not valid sql; DROP???\nrandom text 123"
	require.NoError(t, os.WriteFile(path, []byte("\n"+content+"\n  "), 0600))

	cc := newReportingQueryRunsCreateCmd()
	cc.sqlFile = path

	sql, err := cc.resolveSQL(cc.cmd)
	require.NoError(t, err)
	assert.Equal(t, content, sql, "file contents are passed through as-is (only outer whitespace trimmed)")
}

// --- Unit tests: API base URL validation ---

func TestReportingCreateCmd_InvalidAPIBaseURL(t *testing.T) {
	t.Setenv("STRIPE_API_KEY", "")
	cc := newReportingQueryRunsCreateCmd()
	cc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	cc.rb.APIBaseURL = "http://evil.example.com"
	cc.sql = "SELECT 1"

	err := cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{})
	require.Error(t, err)
}

func TestReportingRetrieveCmd_InvalidAPIBaseURL(t *testing.T) {
	t.Setenv("STRIPE_API_KEY", "")
	rc := newReportingQueryRunsRetrieveCmd()
	rc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	rc.rb.APIBaseURL = "http://evil.example.com"

	err := rc.runReportingQueryRunsRetrieveCmd(rc.cmd, []string{"qryrun_test_123"})
	require.Error(t, err)
}

// --- Integration tests: create HTTP request shape ---

func newTestReportingCreateCmd(t *testing.T, serverURL string) *reportingQueryRunsCreateCmd {
	t.Helper()
	// Ensure the profile API key is used rather than any key set in the
	// environment running the test.
	t.Setenv("STRIPE_API_KEY", "")
	cc := newReportingQueryRunsCreateCmd()
	cc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	cc.rb.APIBaseURL = serverURL
	cc.cmd.SetContext(context.Background())
	return cc
}

func TestReportingCreateCmd_HTTPRequest(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"qr_123","status":"open"}`))
	}))
	defer ts.Close()

	cc := newTestReportingCreateCmd(t, ts.URL)
	cc.sql = "SELECT * FROM charges LIMIT 10"

	err := cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{})
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, http.MethodPost, capturedReq.Method)
	assert.Equal(t, queryRunsPath, capturedReq.URL.Path)
	assert.Equal(t, "Bearer sk_test_1234567890abcdef", capturedReq.Header.Get("Authorization"))
	assert.Equal(t, "application/json", capturedReq.Header.Get("Content-Type"))
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, capturedReq.Header.Get("Stripe-Version"))

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))
	assert.Equal(t, "SELECT * FROM charges LIMIT 10", body["sql"])
}

func TestReportingCreateCmd_HTTPRequest_CompressFile(t *testing.T) {
	var capturedBody []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"qr_123"}`))
	}))
	defer ts.Close()

	cc := newTestReportingCreateCmd(t, ts.URL)
	cc.sql = "SELECT 1"
	cc.compressFile = true

	err := cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{})
	require.NoError(t, err)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(capturedBody, &body))
	resultOptions := body["result_options"].(map[string]interface{})
	assert.Equal(t, true, resultOptions["compress_file"])
}

func TestReportingCreateCmd_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"code":"invalid_sql","type":"invalid_request_error"}}`))
	}))
	defer ts.Close()

	cc := newTestReportingCreateCmd(t, ts.URL)
	cc.sql = "SELECT bad"

	err := cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{})
	require.Error(t, err)

	var reqErr requests.RequestError
	require.ErrorAs(t, err, &reqErr)
	assert.Equal(t, http.StatusBadRequest, reqErr.StatusCode)
	assert.Equal(t, "invalid_sql", reqErr.ErrorCode)
}

// --- Integration tests: retrieve HTTP request shape ---

func newTestReportingRetrieveCmd(t *testing.T, serverURL string) *reportingQueryRunsRetrieveCmd {
	t.Helper()
	t.Setenv("STRIPE_API_KEY", "")
	rc := newReportingQueryRunsRetrieveCmd()
	rc.rb.Profile = &config.Profile{APIKey: "sk_test_1234567890abcdef"}
	rc.rb.APIBaseURL = serverURL
	rc.cmd.SetContext(context.Background())
	return rc
}

func TestReportingRetrieveCmd_HTTPRequest(t *testing.T) {
	var capturedReq *http.Request

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"qr_123","status":"completed","result":{"download_url":"https://example.com/f"}}`))
	}))
	defer ts.Close()

	rc := newTestReportingRetrieveCmd(t, ts.URL)

	err := rc.runReportingQueryRunsRetrieveCmd(rc.cmd, []string{"qr_123"})
	require.NoError(t, err)
	require.NotNil(t, capturedReq)

	assert.Equal(t, http.MethodGet, capturedReq.Method)
	assert.Equal(t, queryRunsPath+"/qr_123", capturedReq.URL.Path)
	assert.Equal(t, "Bearer sk_test_1234567890abcdef", capturedReq.Header.Get("Authorization"))
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, capturedReq.Header.Get("Stripe-Version"))
}

// --- Unit tests: command construction ---

func TestNewReportingQueryRunsCreateCmd_IsPreview(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	assert.True(t, cc.rb.IsPreviewCommand, "query-runs create must use the preview Stripe-Version header")
	assert.Equal(t, http.MethodPost, cc.rb.Method)
}

func TestNewReportingQueryRunsRetrieveCmd_IsPreview(t *testing.T) {
	rc := newReportingQueryRunsRetrieveCmd()
	assert.True(t, rc.rb.IsPreviewCommand, "query-runs retrieve must use the preview Stripe-Version header")
	assert.Equal(t, http.MethodGet, rc.rb.Method)
}

func TestReportingCmd_CommandPaths(t *testing.T) {
	rc := newReportingCmd()

	createCmd, _, err := rc.cmd.Find([]string{"query-runs", "create"})
	require.NoError(t, err)
	assert.Equal(t, "reporting query-runs create", createCmd.CommandPath())

	retrieveCmd, _, err := rc.cmd.Find([]string{"query-runs", "retrieve"})
	require.NoError(t, err)
	assert.Equal(t, "reporting query-runs retrieve", retrieveCmd.CommandPath())
}

func TestNewReportingQueryRunsCreateCmd_Flags(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()

	require.NotNil(t, cc.cmd.Flags().Lookup("sql"))
	require.NotNil(t, cc.cmd.Flags().Lookup("sql-file"))
	require.NotNil(t, cc.cmd.Flags().Lookup("compress-file"))
}

func TestNewReportingCmd_NotHidden(t *testing.T) {
	rc := newReportingCmd()
	assert.False(t, rc.cmd.Hidden, "reporting command must be listed in help")

	queryRunsCmd, _, err := rc.cmd.Find([]string{"query-runs"})
	require.NoError(t, err)
	assert.False(t, queryRunsCmd.Hidden, "query-runs command must be listed in help")

	createCmd, _, err := rc.cmd.Find([]string{"query-runs", "create"})
	require.NoError(t, err)
	assert.False(t, createCmd.Hidden, "query-runs create must be listed in help")

	retrieveCmd, _, err := rc.cmd.Find([]string{"query-runs", "retrieve"})
	require.NoError(t, err)
	assert.False(t, retrieveCmd.Hidden, "query-runs retrieve must be listed in help")
}

func reportingHelpOutput(t *testing.T, args ...string) string {
	t.Helper()
	root := &cobra.Command{Use: "stripe"}
	root.SetUsageTemplate(getUsageTemplate())
	root.AddCommand(newReportingCmd().cmd)
	out, err := executeCommand(root, args...)
	require.NoError(t, err)
	return out
}

func TestReportingCmd_HelpDescribesPublicPreview(t *testing.T) {
	out := reportingHelpOutput(t, "reporting", "--help")

	assert.Contains(t, out, "Public Preview")
	assert.Contains(t, out, "query-runs")
	assert.Contains(t, out, "Create and retrieve QueryRun objects")
}

func TestReportingQueryRunsCmd_HelpListsCreateAndRetrieve(t *testing.T) {
	out := reportingHelpOutput(t, "reporting", "query-runs", "--help")

	assert.Contains(t, out, "create")
	assert.Contains(t, out, "Create a query run from custom SQL")
	assert.Contains(t, out, "retrieve")
	assert.Contains(t, out, "Retrieve a query run")
}

func TestReportingQueryRunsCreateCmd_HelpDescribesPublicPreview(t *testing.T) {
	out := reportingHelpOutput(t, "reporting", "query-runs", "create", "--help")

	assert.Contains(t, out, "Public Preview")
	assert.Contains(t, out, "/v2/data/reporting/query_runs")
	assert.Contains(t, out, "--compress-file")
	assert.Contains(t, out, "--result_options[compress_file]=true")
	assert.NotContains(t, out, "(sets result_options.compress_file)")

	cc := newReportingQueryRunsCreateCmd()
	usage := cc.cmd.Flags().Lookup("compress-file").Usage
	assert.Contains(t, usage, "API parameter result_options.compress_file")
	assert.Contains(t, usage, "--result_options[compress_file]=true")
	assert.NotContains(t, usage, "(sets ")

	alias := cc.cmd.Flags().Lookup("result_options[compress_file]")
	require.NotNil(t, alias)
	assert.True(t, alias.Hidden, "bracket alias must be hidden so it is not listed as its own flag")
}

func TestReportingQueryRunsRetrieveCmd_HelpDescribesPublicPreview(t *testing.T) {
	out := reportingHelpOutput(t, "reporting", "query-runs", "retrieve", "--help")

	assert.Contains(t, out, "Public Preview")
	assert.Contains(t, out, "/v2/data/reporting/query_runs/{id}")
}

// --- Integration tests: --dry-run is not gated on the active context's mode ---

// profileWithLiveActiveContext returns a profile whose keyring holds an OAK with
// a live active context, so resolving test-mode credentials from it fails with
// ActiveContextLivemodeMismatchError.
func profileWithLiveActiveContext(t *testing.T) *config.Profile {
	t.Helper()
	t.Setenv("STRIPE_API_KEY", "")

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte{}, 0600))

	activeCtxJSON, err := json.Marshal(config.ActiveContext{AccountID: "acct_live123", Livemode: true})
	require.NoError(t, err)

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oak_live_1234567890"),
		config.OAuthActiveContextKeychainKey: activeCtxJSON,
	})
	t.Cleanup(func() {
		config.KeyRing = nil
		viper.Reset()
	})
	(&config.Config{LogLevel: "info", ProfilesFile: profilesFile}).InitConfig()

	return &config.Profile{ProfileName: "default"}
}

func TestReportingCreateCmd_DryRunIgnoresActiveContextMode(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	cc.rb.Profile = profileWithLiveActiveContext(t)
	cc.rb.APIBaseURL = stripe.DefaultAPIBaseURL
	cc.rb.DryRun = true
	cc.sql = "SELECT * FROM charges LIMIT 10"

	var out bytes.Buffer
	cc.cmd.SetOut(&out)

	// A dry run sends nothing, so the live/sandbox gate must not stop you
	// inspecting the request it would have sent.
	require.NoError(t, cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{}))

	var output requests.DryRunOutput
	require.NoError(t, json.Unmarshal(out.Bytes(), &output))
	assert.Equal(t, http.MethodPost, output.DryRun.Method)
	assert.Equal(t, stripe.DefaultAPIBaseURL+queryRunsPath, output.DryRun.URL)
	assert.Equal(t, "SELECT * FROM charges LIMIT 10", output.DryRun.Params["sql"])
	// Falls back to the mode that is actually active, so the preview reflects
	// what would really be sent.
	assert.Equal(t, "true", output.DryRun.Headers["Stripe-Livemode"])
}

func TestReportingCreateCmd_RealRequestStillGatedOnActiveContextMode(t *testing.T) {
	cc := newReportingQueryRunsCreateCmd()
	cc.rb.Profile = profileWithLiveActiveContext(t)
	cc.rb.APIBaseURL = stripe.DefaultAPIBaseURL
	cc.sql = "SELECT * FROM charges LIMIT 10"

	err := cc.runReportingQueryRunsCreateCmd(cc.cmd, []string{})
	require.Error(t, err, "leniency must be scoped to --dry-run")
	assert.Contains(t, err.Error(), "--live")
}

// --- Unit tests: merging query-runs into the generated `reporting` namespace ---

// generatedReportingNamespace mimics what resources_gen.go registers: a bare
// namespace command with no descriptions, hosting the Sigma resources.
func generatedReportingNamespace() *cobra.Command {
	ns := &cobra.Command{Use: "reporting"}
	reportRuns := &cobra.Command{Use: "report_runs"}
	reportRuns.AddCommand(&cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}})
	ns.AddCommand(reportRuns)
	ns.AddCommand(&cobra.Command{Use: "report_types"})
	return ns
}

func TestAddReportingQueryRunsCmd_MergesIntoGeneratedNamespace(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	root.AddCommand(generatedReportingNamespace())

	addReportingQueryRunsCmd(root)

	var reportingCmds []*cobra.Command
	for _, cmd := range root.Commands() {
		if cmd.Name() == "reporting" {
			reportingCmds = append(reportingCmds, cmd)
		}
	}
	require.Len(t, reportingCmds, 1, "two root children named reporting shadow each other")

	names := make([]string, 0, 3)
	for _, child := range reportingCmds[0].Commands() {
		names = append(names, child.Name())
	}
	assert.ElementsMatch(t, []string{"query-runs", "report_runs", "report_types"}, names)
}

func TestAddReportingQueryRunsCmd_SigmaResourcesStayReachable(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	root.AddCommand(generatedReportingNamespace())

	addReportingQueryRunsCmd(root)

	// The bug this guards: the shadowing sibling had no report_runs child and no
	// Run function, so cobra resolved this to the wrong command, printed help and
	// exited 0.
	listCmd, _, err := root.Find([]string{"reporting", "report_runs", "list"})
	require.NoError(t, err)
	assert.Equal(t, "stripe reporting report_runs list", listCmd.CommandPath())

	createCmd, _, err := root.Find([]string{"reporting", "query-runs", "create"})
	require.NoError(t, err)
	assert.Equal(t, "stripe reporting query-runs create", createCmd.CommandPath())
}

func TestAddReportingQueryRunsCmd_DescribesTheMergedNamespace(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	root.AddCommand(generatedReportingNamespace())

	addReportingQueryRunsCmd(root)

	ns, _, err := root.Find([]string{"reporting"})
	require.NoError(t, err)
	assert.Equal(t, reportingNamespaceShort, ns.Short, "--map lists Short; an empty one leaves the namespace undescribed")
	assert.Contains(t, ns.Long, "query-runs")
	assert.Contains(t, ns.Long, "report_runs")
}

func TestAddReportingQueryRunsCmd_KeepsGeneratedDescriptions(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	ns := generatedReportingNamespace()
	ns.Short = "Generated short"
	ns.Long = "Generated long"
	root.AddCommand(ns)

	addReportingQueryRunsCmd(root)

	assert.Equal(t, "Generated short", ns.Short, "do not overwrite descriptions the spec supplies")
	assert.Equal(t, "Generated long", ns.Long)
}

func TestAddReportingQueryRunsCmd_FallsBackToTopLevel(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}

	addReportingQueryRunsCmd(root)

	createCmd, _, err := root.Find([]string{"reporting", "query-runs", "create"})
	require.NoError(t, err)
	assert.Equal(t, "stripe reporting query-runs create", createCmd.CommandPath())
}

func TestReportingQueryRunsCmd_ExampleIndentationMatchesTemplate(t *testing.T) {
	// query-runs hangs off the generated namespace, whose template renders
	// {{.Example}} unindented. Its own template must supply the indentation, and
	// the Example's first line must not double up on it.
	root := &cobra.Command{Use: "stripe"}
	root.AddCommand(generatedReportingNamespace())
	addReportingQueryRunsCmd(root)

	out, err := executeCommand(root, "reporting", "query-runs", "create", "--help")
	require.NoError(t, err)

	lines := strings.Split(out, "\n")
	i := indexOf(lines, "Examples:")
	require.GreaterOrEqual(t, i, 0, "help should have an Examples section:\n%s", out)
	require.Less(t, i+1, len(lines))
	assert.Equal(t, "  # Run an ad hoc query", lines[i+1], "example lines should be indented exactly two spaces")
}
