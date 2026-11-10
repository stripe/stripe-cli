package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

func captureManagementDiagnostics(t *testing.T, fn func()) (string, string) {
	t.Helper()
	logger := log.StandardLogger()
	oldOutput, oldLevel, oldFormatter, oldStderr := logger.Out, logger.Level, logger.Formatter, os.Stderr
	var debug bytes.Buffer
	log.SetOutput(&debug)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() {
		log.SetOutput(oldOutput)
		log.SetLevel(oldLevel)
		log.SetFormatter(oldFormatter)
		os.Stderr = oldStderr
		reader.Close()
	}()
	fn()
	require.NoError(t, writer.Close())
	headers, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(headers), debug.String()
}

func TestManagementDiagnosticsFailures(t *testing.T) {
	for _, test := range []struct {
		name, operation, path, body, field string
		status                             int
	}{
		{"playground HTTP", "create", "/v1/stripecli/playground_context", `{"error":{"code":"permission_denied","message":"body_secret"}}`, "", 403},
		{"workspace HTTP", "list", "/v1/stripecli/workspace_context", `{"error":{"code":"permission_denied","message":"body_secret"}}`, "", 403},
		{"playground decode", "create", "/v1/stripecli/playground_context", `{"playground_id":`, "", 200},
		{"workspace ID", "list", "/v1/stripecli/workspace_context", `{"workspace_id":"body_secret"}`, "", 200},
		{"create HTTP", "create", "/v2/sandboxes", `{"error":{"code":"compartments_max_tree_size_reached","message":"body_secret"}}`, "", 400},
		{"create validation", "create", "/v2/sandboxes", `{"v1_account_id":"body_secret"}`, "", 200},
		{"list HTTP", "list", accessibleSandboxesPath, `{"error":{"code":"permission_denied","message":"body_secret"}}`, "", 403},
		{"list decode", "list", accessibleSandboxesPath, `{"workspaces":`, "", 200},
		{"list ID", "list", accessibleSandboxesPath, `{"workspaces":[{"id":"wksp_test_ok","merchant_id":"acct_ok","name":"good"},{"id":"body_secret","merchant_id":"acct_ok","name":"name_secret"}]}`, "workspaces[1]: id", 200},
		{"list nested account", "list", accessibleSandboxesPath, `{"organizations":[{"workspaces":[{"id":"wksp_test_ok","merchant_id":"body_secret","name":"name_secret"}]}]}`, "organizations[0].workspaces[0]: merchant_id", 200},
		{"list wrong name type", "list", accessibleSandboxesPath, `{"organizations":[{"workspaces":[{"id":"wksp_test_ok","merchant_id":"acct_ok","name":{"secret":"body_secret"}}]}]}`, "organizations[0].workspaces[0]: name has an invalid JSON type", 200},
		{"list name", "list", accessibleSandboxesPath, `{"workspaces":[{"id":"wksp_test_ok","merchant_id":"acct_ok","name":" "}]}`, "workspaces[0]: name", 200},
		{"delete HTTP", "delete", "/v2/workspaces/undocumented/testmode/wksp_test_target/close", `{"error":{"code":"permission_denied","message":"body_secret"}}`, "", 403},
		{"delete validation", "delete", "/v2/workspaces/undocumented/testmode/wksp_test_target/close", `{"id":"body_secret"}`, "", 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_parent", true, "oak_secret")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == test.path {
					w.Header().Set("Request-Id", "req_failure123")
					w.WriteHeader(test.status)
					_, _ = w.Write([]byte(test.body))
					return
				}
				switch r.URL.Path {
				case "/v1/stripecli/playground_context":
					_, _ = w.Write([]byte(`{"playground_id":"play_parent"}`))
				case "/v1/stripecli/workspace_context":
					_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
				case accessibleSandboxesPath:
					_, _ = w.Write([]byte(`{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"name_secret"}]}`))
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
				}
			}))
			defer server.Close()
			client := NewManagementClient(server.URL, profile)
			var resultErr error
			_, debug := captureManagementDiagnostics(t, func() {
				switch test.operation {
				case "create":
					_, resultErr = client.Create(context.Background(), CreateOptions{Name: "request_secret"})
				case "list":
					_, resultErr = client.ListAccessible(context.Background())
				case "delete":
					_, resultErr = client.Delete(context.Background(), "acct_target")
				}
			})
			require.ErrorContains(t, resultErr, "(request ID: req_failure123)")
			if test.field != "" {
				require.ErrorContains(t, resultErr, test.field)
			}
			category, ok := errorcategory.Get(resultErr)
			require.True(t, ok)
			require.NotEmpty(t, category)
			require.Contains(t, debug, `"request_id":"req_failure123"`)
			require.Contains(t, debug, fmt.Sprintf(`"status":%d`, test.status))
			for _, secret := range []string{"body_secret", "request_secret", "name_secret", "oak_secret"} {
				require.NotContains(t, debug+resultErr.Error(), secret)
			}
		})
	}
}

func TestManagementDiagnosticsRefreshFailure(t *testing.T) {
	profile := managementTestProfile(t, "acct_parent", true, "oak_initial")
	config.OAuthTokenRefresher = func(*config.Profile) error { return errors.New("body_secret refresh_token_secret") }
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Request-Id", "req_refresh123")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"body_secret"}}`))
	}))
	defer server.Close()
	_, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
	require.ErrorContains(t, err, "request ID: req_refresh123")
	require.NotContains(t, err.Error(), "body_secret")
	require.NotContains(t, err.Error(), "refresh_token_secret")
}

func assertRetryDiagnostics(t *testing.T, headers, debug, stage, effectiveContext, mode string) {
	t.Helper()
	require.Equal(t, 2, strings.Count(headers, "> POST "))
	var attempts []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(debug), "\n") {
		var fields map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(line), &fields))
		if fields["stage"] == stage {
			attempts = append(attempts, fields)
		}
	}
	require.Len(t, attempts, 2)
	require.Equal(t, float64(401), attempts[0]["status"])
	require.Equal(t, "unauthorized", attempts[0]["error_code"])
	require.Equal(t, float64(200), attempts[1]["status"])
	for _, fields := range attempts {
		require.Equal(t, effectiveContext, fields["stripe_context"])
		require.Equal(t, mode, fields["livemode"])
		require.Equal(t, "req_retry123", fields["request_id"])
	}
	require.NotContains(t, headers+debug, "oak_initial")
	require.NotContains(t, headers+debug, "oak_refreshed")
}
