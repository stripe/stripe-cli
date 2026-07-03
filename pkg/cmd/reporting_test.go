package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/requests"
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
