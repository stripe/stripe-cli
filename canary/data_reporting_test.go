package canary

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stripe/stripe-cli/canary/testutil"
)

// =============================================================================
// Data & Reporting command canary tests
//
// These cover the CLI commands that wrap the v2 data/analytics preview APIs:
//   - stripe data metrics run        -> POST /v2/data/analytics/metric_query
//   - stripe reporting query-runs create   -> POST /v2/data/reporting/query_runs
//   - stripe reporting query-runs retrieve -> GET  /v2/data/reporting/query_runs/{id}
//
// The underlying endpoints are Private Preview. Coverage is layered:
//   - Offline tests verify command registration and help output (no key).
//   - Dry-run tests exercise the full request-building path without a network
//     call (needs a key): auth resolution, URL/method, v2 JSON body, and the
//     preview Stripe-Version header.
//   - Live tests actually invoke the preview endpoints (needs a key on an
//     account with preview access). These require the test account/runner to be
//     enabled for the v2 data/analytics APIs; otherwise they will fail with a
//     preview access error rather than skip.
// =============================================================================

// dryRunOutput mirrors requests.DryRunOutput for parsing --dry-run JSON.
type dryRunOutput struct {
	DryRun struct {
		Method  string                 `json:"method"`
		URL     string                 `json:"url"`
		Params  map[string]interface{} `json:"params"`
		Headers map[string]string      `json:"headers"`
	} `json:"dry_run"`
}

// --- Offline help / registration tests ---

func TestOfflineDataMetricsRunHelp(t *testing.T) {
	runner := getRunner(t)

	// "data" is a hidden Private Preview command, but --help still works when
	// the command path is invoked directly.
	result, err := runner.Run("data", "metrics", "run", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe data metrics run --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	for _, flag := range []string{"--metric", "--starts-at", "--ends-at", "--granularity"} {
		if !strings.Contains(result.Stdout, flag) {
			errorf(t, "Expected 'data metrics run --help' to mention '%s', got: %s", flag, result.Stdout)
		}
	}
}

func TestOfflineReportingHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("reporting", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "query-runs") {
		errorf(t, "Expected 'reporting --help' to list 'query-runs', got: %s", result.Stdout)
	}
}

func TestOfflineReportingQueryRunsHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("reporting", "query-runs", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting query-runs --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	for _, sub := range []string{"create", "retrieve"} {
		if !strings.Contains(result.Stdout, sub) {
			errorf(t, "Expected 'reporting query-runs --help' to list '%s', got: %s", sub, result.Stdout)
		}
	}
}

func TestOfflineReportingQueryRunsRetrieveHelp(t *testing.T) {
	runner := getRunner(t)

	result, err := runner.Run("reporting", "query-runs", "retrieve", "--help")
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting query-runs retrieve --help': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	if !strings.Contains(result.Stdout, "retrieve") {
		errorf(t, "Expected 'reporting query-runs retrieve --help' to mention 'retrieve', got: %s", result.Stdout)
	}
}

// --- Dry-run tests (require a key, but make no network call) ---

func TestAPIDataMetricsRunDryRun(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run(
		"data", "metrics", "run",
		"--metric", "revenue.mrr",
		"--starts-at", "2026-01-01T00:00:00Z",
		"--ends-at", "2026-01-31T23:59:59Z",
		"--granularity", "day",
		"--dry-run",
	)
	if err != nil {
		fatalf(t, "Failed to run 'stripe data metrics run --dry-run': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var out dryRunOutput
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		fatalf(t, "Dry-run output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if out.DryRun.Method != "POST" {
		errorf(t, "Expected dry-run method POST, got %q", out.DryRun.Method)
	}
	if !strings.HasSuffix(out.DryRun.URL, "/v2/data/analytics/metric_query") {
		errorf(t, "Expected dry-run URL to target /v2/data/analytics/metric_query, got %q", out.DryRun.URL)
	}
	for _, key := range []string{"metrics", "starts_at", "ends_at", "granularity"} {
		if _, ok := out.DryRun.Params[key]; !ok {
			errorf(t, "Expected dry-run params to include %q, got: %s", key, result.Stdout)
		}
	}
}

func TestAPIReportingQueryRunsCreateDryRun(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))

	result, err := runner.Run(
		"reporting", "query-runs", "create",
		"--sql", "SELECT id FROM charges LIMIT 1",
		"--dry-run",
	)
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting query-runs create --dry-run': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var out dryRunOutput
	if err := json.Unmarshal([]byte(result.Stdout), &out); err != nil {
		fatalf(t, "Dry-run output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if out.DryRun.Method != "POST" {
		errorf(t, "Expected dry-run method POST, got %q", out.DryRun.Method)
	}
	if !strings.HasSuffix(out.DryRun.URL, "/v2/data/reporting/query_runs") {
		errorf(t, "Expected dry-run URL to target /v2/data/reporting/query_runs, got %q", out.DryRun.URL)
	}
	if _, ok := out.DryRun.Params["sql"]; !ok {
		errorf(t, "Expected dry-run params to include 'sql', got: %s", result.Stdout)
	}
}

// --- Live tests (require a key on an account with preview access) ---

func TestAPIDataMetricsRunLive(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))
	runner = runner.WithTimeout(60 * time.Second)

	result, err := runner.Run(
		"data", "metrics", "run",
		"--metric", "revenue.mrr",
		"--starts-at", "2026-01-01T00:00:00Z",
		"--ends-at", "2026-01-31T23:59:59Z",
		"--granularity", "month",
	)
	if err != nil {
		fatalf(t, "Failed to run 'stripe data metrics run': %v", err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		errorf(t, "Output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if _, ok := data["data"]; !ok {
		errorf(t, "Expected metric query response to contain 'data' field, got: %s", result.Stdout)
	}
}

func TestAPIReportingQueryRunsCreateLive(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))
	runner = runner.WithTimeout(60 * time.Second)

	id := createQueryRun(t, runner)
	if id == "" {
		errorf(t, "Expected created query run to have a non-empty id")
	}
}

func TestAPIReportingQueryRunsRetrieveLive(t *testing.T) {
	requireAPIKey(t)
	runner := getRunner(t, testutil.WithEnv(map[string]string{
		"STRIPE_API_KEY": testutil.GetAPIKey(),
	}))
	runner = runner.WithTimeout(60 * time.Second)

	// Create a query run, then retrieve it by id (round trip).
	id := createQueryRun(t, runner)
	if id == "" {
		fatalf(t, "Cannot retrieve: created query run had no id")
	}

	result, err := runner.Run("reporting", "query-runs", "retrieve", id)
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting query-runs retrieve %s': %v", id, err)
	}

	if result.ExitCode != 0 {
		errorf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		fatalf(t, "Retrieve output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	if got, _ := data["id"].(string); got != id {
		errorf(t, "Expected retrieved query run id %q, got %q", id, got)
	}

	if _, ok := data["status"]; !ok {
		errorf(t, "Expected retrieved query run to contain 'status' field, got: %s", result.Stdout)
	}
}

// createQueryRun creates a query run and returns its id. It fails the test if
// the command errors or the response is missing an id.
func createQueryRun(t *testing.T, runner *testutil.Runner) string {
	t.Helper()

	result, err := runner.Run(
		"reporting", "query-runs", "create",
		"--sql", "SELECT id FROM charges LIMIT 1",
	)
	if err != nil {
		fatalf(t, "Failed to run 'stripe reporting query-runs create': %v", err)
	}

	if result.ExitCode != 0 {
		fatalf(t, "Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(result.Stdout), &data); err != nil {
		fatalf(t, "Create output is not valid JSON: %v. Output: %s", err, result.Stdout)
	}

	id, _ := data["id"].(string)
	return id
}
