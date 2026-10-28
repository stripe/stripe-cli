package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/plugins"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
	"github.com/stripe/stripe-cli/pkg/version"
)

type failOnRead struct{ t *testing.T }

func (r failOnRead) Read([]byte) (int, error) {
	r.t.Fatal("recovery must not read stdin")
	return 0, io.EOF
}

func TestInstallRecoveryEOFDoesNotConsent(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()
	cfg.Profile.APIKey = ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	ic := NewInstallCmd(cfg)
	ic.fs = fs
	ic.apiBaseURL = server.URL
	ic.stdinIsTerminal = func() bool { return true }
	ic.lookupEnvFn = func(string) string { return "" }
	ic.loginFn = func(context.Context) error { t.Fatal("unexpected login"); return nil }
	ic.Cmd.SetIn(strings.NewReader(""))
	ic.Cmd.SetErr(io.Discard)
	ic.Cmd.SetContext(context.Background())
	require.EqualError(t, ic.runInstallCmd(ic.Cmd, []string{"example"}), "login canceled")
}

func TestInstallRecoveryRetainsRetryFailure(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"not found", http.StatusNotFound, `{"error":{"message":"not found"}}`, "could not find an available release"},
		{"server auth", http.StatusUnauthorized, `{"error":{"message":"invalid credentials"}}`, "invalid credentials"},
		{"server error", http.StatusInternalServerError, `{"error":{"message":"metadata unavailable"}}`, "metadata unavailable"},
		{"minimum version", http.StatusBadRequest, `{"error":{"code":"plugin_requires_newer_cli","message":"upgrade required","min_core_version":"99.0.0"}}`, "99.0.0"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, fs, cleanup := setupPluginCommandTest(t)
			defer cleanup()
			key := cfg.Profile.APIKey
			cfg.Profile.APIKey = ""
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/ajax/stripecli/plugins_metadata" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				require.Equal(t, "/v1/stripecli/get-plugin-metadata", r.URL.Path)
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			ic := NewInstallCmd(cfg)
			ic.fs = fs
			ic.apiBaseURL = server.URL
			ic.stdinIsTerminal = func() bool { return true }
			ic.lookupEnvFn = func(string) string { return "" }
			logins := 0
			ic.loginFn = func(context.Context) error { logins++; cfg.Profile.APIKey = key; return nil }
			ic.Cmd.SetIn(strings.NewReader("\n"))
			ic.Cmd.SetErr(io.Discard)
			ic.Cmd.SetContext(context.Background())

			err := ic.runInstallCmd(ic.Cmd, []string{"example"})
			require.ErrorContains(t, err, test.want)
			require.Equal(t, 1, logins)
			require.Equal(t, 2, requestCount)
			require.NotErrorIs(t, err, validators.ErrAPIKeyNotConfigured)
			if test.name == "minimum version" {
				var newer *plugins.ErrPluginRequiresNewerCLI
				require.ErrorAs(t, err, &newer)
			} else if test.status != http.StatusNotFound {
				var requestErr requests.RequestError
				require.ErrorAs(t, err, &requestErr)
				require.Equal(t, test.status, requestErr.StatusCode)
			}
		})
	}
}

func TestInstallFailuresDoNotOfferLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/ajax/stripecli/plugins_metadata", r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"metadata unavailable"}}`)
	}))
	defer server.Close()
	closedServer := httptest.NewServer(http.NotFoundHandler())
	closedServer.Close()

	for _, kind := range []string{"invalid override", "refresh", "transport", "server", "cancellation"} {
		t.Run(kind, func(t *testing.T) {
			cfg, fs, cleanup := setupPluginCommandTest(t)
			defer cleanup()
			refreshErr := errors.New("session refresh failed")
			cfg.Profile.APIKey = ""
			cfg.Profile.AccountID = "saved-account"
			switch kind {
			case "invalid override":
				cfg.Profile.APIKey = "malformed"
			case "refresh":
				require.NoError(t, config.KeyRing.Set(cfg.Profile.KeyringKey(config.UATKeychainItemKey), []byte("oak_test_fixture"), "test token"))
				require.NoError(t, cfg.Profile.SaveUATExpiresAt(time.Now().Add(-time.Hour)))
				previous := config.OAuthTokenRefresher
				t.Cleanup(func() { config.OAuthTokenRefresher = previous })
				config.OAuthTokenRefresher = func(*config.Profile) error { return refreshErr }
			}
			ic := NewInstallCmd(cfg)
			ic.fs = fs
			ic.apiBaseURL = server.URL
			if kind == "transport" {
				ic.apiBaseURL = closedServer.URL
			}
			ic.stdinIsTerminal = func() bool { return true }
			ic.lookupEnvFn = func(string) string { return "" }
			ic.Cmd.SetIn(failOnRead{t})
			var output strings.Builder
			ic.Cmd.SetErr(&output)
			ic.loginFn = func(context.Context) error { t.Fatal("unexpected login"); return nil }
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if kind == "cancellation" {
				cancel()
			}
			ic.Cmd.SetContext(ctx)
			err := ic.runInstallCmd(ic.Cmd, []string{"example"})
			switch kind {
			case "refresh":
				require.ErrorIs(t, err, refreshErr)
			case "invalid override":
				require.ErrorContains(t, err, "too short")
			case "transport":
				var transportErr *url.Error
				require.ErrorAs(t, err, &transportErr)
			case "server":
				var requestErr requests.RequestError
				require.ErrorAs(t, err, &requestErr)
				require.Equal(t, http.StatusInternalServerError, requestErr.StatusCode)
			case "cancellation":
				require.ErrorIs(t, err, context.Canceled)
			}
			require.NotErrorIs(t, err, validators.ErrAPIKeyNotConfigured)
			require.Empty(t, output.String())
			require.NotContains(t, err.Error(), "stripe login")
		})
	}
}

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
	cfg.Profile.AccountID = "saved-account"

	server := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/ajax/stripecli/plugins_metadata", req.URL.Path)
		res.WriteHeader(http.StatusNotFound)
		_, _ = res.Write([]byte(`{"error":{"message":"not found"}}`))
	}))
	defer server.Close()

	for _, agent := range []bool{false, true} {
		t.Run(fmt.Sprintf("agent=%t", agent), func(t *testing.T) {
			ic := NewInstallCmd(cfg)
			ic.fs = fs
			ic.apiBaseURL = server.URL
			ic.stdinIsTerminal = func() bool { return agent }
			ic.lookupEnvFn = func(key string) string {
				if agent && key == "CLAUDECODE" {
					return "1"
				}
				return ""
			}
			ic.loginFn = func(context.Context) error { t.Fatal("unexpected login"); return nil }
			ic.Cmd.SetIn(failOnRead{t})
			ic.Cmd.SetContext(context.Background())

			err := ic.runInstallCmd(ic.Cmd, []string{"nonexistent-plugin"})
			require.ErrorIs(t, err, validators.ErrAPIKeyNotConfigured)
			category, ok := errorcategory.Get(err)
			require.True(t, ok)
			require.Equal(t, errorcategory.Auth, category)
			require.ErrorContains(t, err, "stripe login")
			require.ErrorContains(t, err, "stripe whoami")
			require.ErrorContains(t, err, "retry")
		})
	}
}

// TestRunInstallCmdPluginRequiresNewerCLI ensures version requirements bypass
// authentication recovery and available cached metadata.
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
	require.Contains(t, err.Error(), "could not find an available release")
	require.Contains(t, err.Error(), "nonexistent-plugin")
	require.Contains(t, err.Error(), "plugin name")
	require.Contains(t, err.Error(), "stripe whoami")
	require.Contains(t, err.Error(), "active account")
}

func TestPrintUnrequestedDowngradeNote(t *testing.T) {
	originalVersion := version.Version
	version.Version = "1.20.0"
	defer func() { version.Version = originalVersion }()

	tests := []struct {
		name        string
		prevVersion string
		newVersion  string
		expectNote  bool
	}{
		{name: "rollback to an older release", prevVersion: "2.0.0", newVersion: "1.5.0", expectNote: true},
		{name: "ordinary upgrade", prevVersion: "1.5.0", newVersion: "2.0.0"},
		{name: "same version", prevVersion: "2.0.0", newVersion: "2.0.0"},
		{name: "unparseable versions", prevVersion: "not-a-version", newVersion: "1.5.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out strings.Builder
			printUnrequestedDowngradeNote(&out, "appA", test.prevVersion, test.newVersion)

			if !test.expectNote {
				require.Empty(t, out.String())
				return
			}

			require.Contains(t, out.String(), "v1.5.0 is the newest appA release available to Stripe CLI 1.20.0")
			require.Contains(t, out.String(), "Newer releases may require a newer Stripe CLI")
			require.Contains(t, out.String(), "https://docs.stripe.com/stripe-cli/upgrade")
		})
	}
}
