package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

func captureManagementDiagnostics(t *testing.T, level log.Level, fn func()) (string, string) {
	t.Helper()
	logger := log.StandardLogger()
	oldOutput, oldLevel, oldFormatter, oldStderr := logger.Out, logger.Level, logger.Formatter, os.Stderr
	var debug bytes.Buffer
	log.SetOutput(&debug)
	log.SetLevel(level)
	log.SetFormatter(&log.JSONFormatter{})
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = writer
	defer func() {
		log.SetOutput(oldOutput)
		log.SetLevel(oldLevel)
		log.SetFormatter(oldFormatter)
		os.Stderr = oldStderr
		writer.Close()
		reader.Close()
	}()
	fn()
	require.NoError(t, writer.Close())
	headers, err := io.ReadAll(reader)
	require.NoError(t, err)
	return string(headers), debug.String()
}

func TestManagementDiagnosticsControls(t *testing.T) {
	for _, test := range []struct {
		name           string
		level          log.Level
		headers, debug bool
	}{
		{"default", log.InfoLevel, false, false},
		{"headers", log.InfoLevel, true, false},
		{"debug", log.DebugLevel, false, true},
		{"both", log.DebugLevel, true, true},
		{"trace", log.TraceLevel, false, true},
	} {
		for _, forbidden := range []bool{false, true} {
			name := "success"
			if forbidden {
				name = "forbidden"
			}
			t.Run(test.name+"/"+name, func(t *testing.T) {
				profile := managementTestProfile(t, "acct_parent", true, "oak_secret")
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Request-Id", "req_fixture123")
					w.Header().Set("X-Private", "header_secret")
					switch r.URL.Path {
					case "/v1/stripecli/workspace_context":
						if forbidden {
							w.WriteHeader(http.StatusForbidden)
							_, _ = w.Write([]byte(`{"error":{"code":"permission_denied","message":"body_secret oak_secret raw_error_secret"}}`))
							return
						}
						_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
					case accessibleSandboxesPath:
						_, _ = w.Write([]byte(`{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"name_secret"}]}`))
					default:
						t.Errorf("unexpected path %s", r.URL.Path)
					}
				}))
				defer server.Close()
				client := NewManagementClient(server.URL, profile)
				client.ShowHeaders = test.headers
				var sandboxes []ManagedSandbox
				var err error

				headers, debug := captureManagementDiagnostics(t, test.level, func() {
					sandboxes, err = client.ListAccessible(context.Background())
				})

				if forbidden {
					require.EqualError(t, err, "could not resolve the active live workspace: the active live account is not authorized for this CLI session; switch context or reauthorize the CLI")
					require.Equal(t, errorcategory.Auth, mustErrorCategory(t, err))
					require.Empty(t, sandboxes)
					require.NotContains(t, err.Error(), "req_fixture123")
				} else {
					require.NoError(t, err)
					require.Equal(t, []ManagedSandbox{{WorkspaceID: "wksp_test_target", AccountID: "acct_target", Name: "name_secret", AccessLevel: SandboxAccessLevelPrivate}}, sandboxes)
				}
				require.Equal(t, test.headers, strings.Contains(headers, "< Request-Id: req_fixture123"))
				if test.headers {
					require.Contains(t, headers, "Bearer [REDACTED]")
					if forbidden {
						require.Contains(t, headers, "< HTTP 403")
					} else {
						require.Contains(t, headers, accessibleSandboxesPath)
					}
				} else {
					require.Empty(t, headers)
				}
				if test.debug {
					lines := strings.Split(strings.TrimSpace(debug), "\n")
					if forbidden {
						require.Len(t, lines, 1)
						assertManagementLog(t, lines[0], "workspace_context", "failure", http.StatusForbidden)
					} else {
						require.Len(t, lines, 2)
						assertManagementLog(t, lines[0], "workspace_context", "success", 0)
						assertManagementLog(t, lines[1], "list", "success", 0)
					}
				} else {
					require.Empty(t, debug)
				}
				for _, secret := range []string{"body_secret", "raw_error_secret", "name_secret", "oak_secret", "header_secret", "permission_denied"} {
					require.NotContains(t, headers+debug, secret)
				}
			})
		}
	}
}

func TestManagementDiagnosticsNetworkFailure(t *testing.T) {
	profile := managementTestProfile(t, "acct_parent", true, "oak_secret")
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	var err error

	headers, debug := captureManagementDiagnostics(t, log.DebugLevel, func() {
		_, err = NewManagementClient(server.URL, profile).ListAccessible(context.Background())
	})

	require.EqualError(t, err, "could not resolve the active live workspace: network request failed")
	require.Equal(t, errorcategory.Network, mustErrorCategory(t, err))
	require.Empty(t, headers)
	assertManagementLog(t, strings.TrimSpace(debug), "workspace_context", "failure", 0)
}

func assertManagementLog(t *testing.T, line, stage, outcome string, status int) {
	t.Helper()
	var fields map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(line), &fields))
	require.NotEmpty(t, fields["time"])
	delete(fields, "time")
	want := map[string]interface{}{
		"level": "debug", "msg": "Sandbox management request completed",
		"operation": "sandbox_list", "stage": stage, "outcome": outcome,
	}
	if status != 0 {
		want["status"] = float64(status)
	}
	require.Equal(t, want, fields)
}

func assertRetryHeaders(t *testing.T, headers, effectiveContext, mode string) {
	t.Helper()
	require.Equal(t, 2, strings.Count(headers, "> POST "))
	require.Equal(t, 2, strings.Count(headers, "< Request-Id: req_retry123"))
	require.Equal(t, 2, strings.Count(headers, "> Stripe-Context: "+effectiveContext))
	require.GreaterOrEqual(t, strings.Count(headers, "> Stripe-Livemode: "+mode), 2)
	require.Contains(t, headers, "< HTTP 401")
	require.Contains(t, headers, "< HTTP 200")
	require.NotContains(t, headers, "oak_initial")
	require.NotContains(t, headers, "oak_refreshed")
}
