package plugin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestParseArg(t *testing.T) {
	// No version
	plugin, version := parseInstallArg("apps")
	require.Equal(t, "apps", plugin)
	require.Equal(t, "", version)

	// Version
	plugin, version = parseInstallArg("apps@2.0.1")
	require.Equal(t, "apps", plugin)
	require.Equal(t, "2.0.1", version)
}

func TestSetInstallTelemetryMetadata(t *testing.T) {
	installCmd := &InstallCmd{}
	metadata := stripe.NewEventMetadata()
	ctx := stripe.WithEventMetadata(context.Background(), metadata)

	installCmd.setInstallTelemetryMetadata(ctx, "apps")

	require.Equal(t, "apps", metadata.PluginName)
}

func TestRunInstallCmdNonExistentPluginNotLoggedIn(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.APIKey = ""
	cfg.Profile.AccountID = ""

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/ajax/stripecli/plugins_metadata":
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
		default:
			res.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Redirect stdin to simulate user typing "cancel" to skip login prompt
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	_, _ = w.WriteString("cancel\n")
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = server.URL
	ic.Cmd.SetContext(context.Background())

	err := ic.runInstallCmd(ic.Cmd, []string{"nonexistent-plugin"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "login canceled")
}

func TestRunInstallCmdNonExistentPluginLoggedIn(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.AccountID = "acct_123"

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/stripecli/get-plugin-metadata":
			res.WriteHeader(http.StatusNotFound)
			_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
		default:
			res.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = server.URL
	ic.Cmd.SetContext(context.Background())

	err := ic.runInstallCmd(ic.Cmd, []string{"nonexistent-plugin"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "no plugin named")
	require.Contains(t, err.Error(), "nonexistent-plugin")
	require.Contains(t, err.Error(), "exists")
}
