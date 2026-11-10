package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestSandboxManagementShowHeaders(t *testing.T) {
	for _, operation := range []string{"create", "list", "delete"} {
		for _, flag := range []string{"--show-headers", "-s"} {
			t.Run(operation+flag, func(t *testing.T) {
				cleanup := setupSandboxTestConfig(t)
				defer cleanup()
				setSandboxCreateOAuthContext(t)
				requestCount := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestCount++
					w.Header().Set("Request-Id", "req_command123")
					switch r.URL.Path {
					case "/v1/stripecli/playground_context":
						_, _ = w.Write([]byte(`{"playground_id":"play_parent"}`))
					case "/v1/stripecli/workspace_context":
						_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
					case "/v2/sandboxes":
						_, _ = w.Write([]byte(`{"v1_account_id":"acct_created"}`))
					case "/v2/compartments/user_accessible_sandboxes":
						_, _ = w.Write([]byte(`{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"}]}`))
					case "/v2/workspaces/undocumented/testmode/wksp_test_target/close":
						_, _ = w.Write([]byte(`{"id":"wksp_test_target"}`))
					default:
						t.Errorf("unexpected path %s", r.URL.Path)
					}
				}))
				defer server.Close()
				var command *cobra.Command
				args := []string{flag, "--api-base", server.URL}
				switch operation {
				case "create":
					command = newSandboxCreateCmd().cmd
					args = append(args, "New sandbox", "--non-interactive")
				case "list":
					command = newSandboxListCmd().cmd
				case "delete":
					command = newSandboxDeleteCmd().cmd
					args = append(args, "acct_target", "--confirm")
				}
				for _, excluded := range []string{"live", "stripe-context", "stripe-account", "data", "verbose"} {
					require.Nil(t, command.Flags().Lookup(excluded))
				}
				var stdout bytes.Buffer
				command.SetOut(&stdout)
				command.SetArgs(args)
				oldStderr := os.Stderr
				reader, writer, err := os.Pipe()
				require.NoError(t, err)
				defer reader.Close()
				defer func() { os.Stderr = oldStderr }()
				os.Stderr = writer
				err = command.Execute()
				require.NoError(t, writer.Close())
				os.Stderr = oldStderr
				headers, readErr := io.ReadAll(reader)
				require.NoError(t, readErr)
				require.NoError(t, err)
				require.Equal(t, requestCount, strings.Count(string(headers), "< Request-Id: req_command123"))
				require.Equal(t, requestCount, strings.Count(string(headers), "> Authorization: Bearer [REDACTED]"))
				require.Contains(t, string(headers), "/v1/stripecli/workspace_context")
				require.NotContains(t, string(headers), "oak_live_test")
				require.NotContains(t, stdout.String(), "Request-Id")
				switch operation {
				case "create":
					require.Contains(t, stdout.String(), "Account ID: acct_created")
					require.Contains(t, string(headers), "> Stripe-Context: play_parent")
				case "list":
					require.Contains(t, stdout.String(), "NAME")
					require.Contains(t, stdout.String(), "acct_target")
				case "delete":
					require.Contains(t, stdout.String(), `Deleted sandbox "Target sandbox" (acct_target)`)
					require.Contains(t, string(headers), "> Stripe-Context: acct_target")
					require.Contains(t, string(headers), "> Stripe-Livemode: false")
				}
			})
		}
	}
}

func TestSandboxCreateRejectsAnonymousShowHeaders(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	command := newSandboxCreateCmd()
	command.cmd.SetArgs([]string{"--email", "user@example.com", "-s"})
	var output bytes.Buffer
	command.cmd.SetOut(&output)
	command.cmd.SetErr(&output)
	require.ErrorContains(t, command.cmd.Execute(), "--show-headers is only valid for managed creation")
	require.Contains(t, command.cmd.Long, "--show-headers requires managed creation")
}
