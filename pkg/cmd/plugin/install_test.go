package plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/version"
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

// TestRunInstallCmdPluginRequiresNewerCLI pins that a version problem is reported
// as one. Not being logged in is the other reason a plugin can fail to resolve, and
// the command cannot tell the two apart from the error alone -- so without the
// typed check it offers to log in, which can never make this CLI new enough.
func TestRunInstallCmdPluginRequiresNewerCLI(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.APIKey = ""
	cfg.Profile.AccountID = ""

	originalVersion := version.Version
	version.Version = "1.29.0"
	defer func() { version.Version = originalVersion }()

	// Cached metadata that could satisfy the install, so the endpoint's refusal is
	// shown to outrank an available fallback rather than being the only thing left.
	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	metadataPath := filepath.Join(configPath, "plugin-metadata", "appA.toml")
	require.NoError(t, fs.MkdirAll(filepath.Dir(metadataPath), 0755))
	require.NoError(t, afero.WriteFile(fs, metadataPath, []byte(fmt.Sprintf(`[[Plugin]]
  Shortname = "appA"
  Binary = "stripe-cli-app-a"
  MagicCookieValue = "APP-A-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.1"
    Sum = "abc123"
`, runtime.GOARCH, runtime.GOOS)), 0644))

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusBadRequest)
		_, _ = res.Write([]byte(`{"error":{"code":"plugin_requires_newer_cli","message":"Version 2.0.1 of the appA plugin requires Stripe CLI 1.30.0 or later.","min_core_version":"1.30.0","param":"version","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	// Left empty on purpose: anything reading stdin here is the login prompt, and it
	// would read EOF and go on to open a browser.
	origStdin := os.Stdin
	r, w, _ := os.Pipe()
	_ = w.Close()
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = server.URL
	ic.dashboardBaseURL = server.URL
	ic.Cmd.SetContext(context.Background())

	err := ic.runInstallCmd(ic.Cmd, []string{"appA@2.0.1"})

	var requiresNewerCLI *plugins.ErrPluginRequiresNewerCLI
	require.ErrorAs(t, err, &requiresNewerCLI)
	require.Equal(t, "1.30.0", requiresNewerCLI.MinCoreVersion)
	require.NotContains(t, err.Error(), "logged in")
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
