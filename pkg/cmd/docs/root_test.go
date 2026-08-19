package docs_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
	"github.com/stripe/stripe-cli/pkg/docs"
	"github.com/stripe/stripe-cli/pkg/docs/markdown"
	"github.com/stripe/stripe-cli/pkg/requests"
)

func TestNew(t *testing.T) {
	root := cmd.New().Root()

	assert.Equal(t, "docs <path>", root.Use)
	assert.NotEmpty(t, root.Short)
}

func TestRootParsesDocsURL(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantPath  string
		wantQuery string
	}{
		{
			name:     "full docs.stripe.com URL",
			args:     []string{"https://docs.stripe.com/connect/accounts"},
			wantPath: "/connect/accounts",
		},
		{
			name:      "full docs.stripe.com URL with query",
			args:      []string{"https://docs.stripe.com/api/customers?api_version=2024-06-30"},
			wantPath:  "/api/customers",
			wantQuery: "api_version=2024-06-30",
		},
		{
			name:     "plain path unchanged",
			args:     []string{"/payments"},
			wantPath: "/payments",
		},
		{
			name:      "plain path with query params",
			args:      []string{"payments?test=foo"},
			wantPath:  "/payments",
			wantQuery: "test=foo",
		},
		{
			name:      "absolute path with query params",
			args:      []string{"/payments?test=foo"},
			wantPath:  "/payments",
			wantQuery: "test=foo",
		},
		{
			name:     "multi-segment args joined",
			args:     []string{"connect", "accounts"},
			wantPath: "/connect/accounts",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotPath, gotQuery string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				gotQuery = r.URL.RawQuery
				fmt.Fprint(w, "# Test\n\nContent.")
			}))
			defer server.Close()

			client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
			renderer, err := markdown.NewRenderer()
			require.NoError(t, err)

			var out bytes.Buffer
			root := cmd.New().WithOptions(
				cmd.WithClient(client),
				cmd.WithRenderer(renderer),
			).Root()
			root.SetOut(&out)
			root.SetArgs(tc.args)

			err = root.ExecuteContext(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tc.wantPath, gotPath)
			assert.Equal(t, tc.wantQuery, gotQuery)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRootRoutesSupportURLs(t *testing.T) {
	tests := []struct {
		name      string
		arg       string
		wantHost  string
		wantPath  string
		wantQuery string
	}{
		{
			name:     "bare support URL",
			arg:      "support.stripe.com/questions/getting-started",
			wantHost: "support.stripe.com",
			wantPath: "/questions/getting-started",
		},
		{
			name:      "schemed support URL with query",
			arg:       "https://support.stripe.com/questions/search?topic=payments&source=cli",
			wantHost:  "support.stripe.com",
			wantPath:  "/questions/search",
			wantQuery: "topic=payments&source=cli",
		},
		{
			name:     "lookalike support subdomain is a docs path",
			arg:      "support.stripe.com.example.com/questions",
			wantPath: "/support.stripe.com.example.com/questions",
		},
		{
			name:     "unsupported host is a docs path",
			arg:      "https://example.com/questions",
			wantPath: "/https://example.com/questions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/markdown")
				fmt.Fprint(w, "# Support")
			}))
			defer server.Close()

			target, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			var gotHost, gotPath, gotQuery string
			httpClient := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				gotHost = req.URL.Host
				gotPath = req.URL.Path
				gotQuery = req.URL.RawQuery

				forwarded := req.Clone(req.Context())
				forwarded.URL.Scheme = target.URL.Scheme
				forwarded.URL.Host = target.URL.Host
				return http.DefaultTransport.RoundTrip(forwarded)
			})}

			client := docs.NewClient("test").WithOptions(
				docs.WithBaseURL(server.URL),
				docs.WithHTTPClient(httpClient),
			)
			renderer, err := markdown.NewRenderer()
			require.NoError(t, err)

			root := cmd.New().WithOptions(
				cmd.WithClient(client),
				cmd.WithRenderer(renderer),
			).Root()
			root.SetOut(io.Discard)
			root.SetArgs([]string{"--non-interactive", "--no-pager", tt.arg})

			require.NoError(t, root.ExecuteContext(context.Background()))
			wantHost := tt.wantHost
			if wantHost == "" {
				wantHost = target.URL.Host
			}
			assert.Equal(t, wantHost, gotHost)
			assert.Equal(t, tt.wantPath, gotPath)
			assert.Equal(t, tt.wantQuery, gotQuery)
		})
	}
}

func TestRootPrefixesPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/connect/accounts", r.URL.Path)
		fmt.Fprint(w, "# Accounts\n\nManage connected accounts.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"connect", "accounts"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Accounts")
}

func TestFetchPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept payments with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
}

// TestAgentDetectionDisablesTUI verifies that when an agent env var is set,
// running with no arguments prints help rather than launching the interactive
// TUI. In a real terminal (where term.IsTerminal is true) this would otherwise
// open BubbleTea; the test environment uses a bytes.Buffer so it also confirms
// no panic or hang occurs through the non-TUI code path.
func TestAgentDetectionDisablesTUI(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	var out bytes.Buffer
	root := cmd.New().Root()
	root.SetOut(&out)
	root.SetArgs([]string{})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Usage:", "expected help output, not TUI")
}

func TestAgentDetectionForcesNottyStyle(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.NotContains(t, result, "\x1b[", "should not contain ANSI escape codes when agent is detected")
}

func TestColorOffForcesNottyStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	cfg := &cliconfig.Config{Color: "off"}

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(cfg),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.NotContains(t, result, "\x1b[", "should not contain ANSI escape codes when --color=off")
}

func TestColorOnForcesColorEvenWithAgent(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	cfg := &cliconfig.Config{Color: "on"}

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(cfg),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.Contains(t, result, "\x1b[", "should contain ANSI escape codes when --color=on even with agent")
}

func TestPreRun_LoggerRespectsConfiguredLevel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "# Test\n\nContent.")
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	testLogger := log.New()
	testLogger.SetOutput(&logBuf)
	testLogger.SetLevel(log.DebugLevel)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
		cmd.WithLogger(log.NewEntry(testLogger)),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/test"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, logBuf.String(), "injected debug-level logger should capture log output")
}

func TestPrefsForwardedToPageFetch(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()
	require.NoError(t, cfg.Profile.WriteConfigField("docs_prefs.lang", "ruby"))

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, "# File Upload")
	}))
	defer server.Close()

	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	root := cmd.New().WithOptions(
		cmd.WithConfig(&cliconfig.Config{Color: "off", Profile: cfg.Profile}),
		cmd.WithClient(docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"--non-interactive", "/file-upload"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, gotQuery, "lang=ruby")
}

func TestPrefsNotOverriddenByExistingQueryParam(t *testing.T) {
	cfg, cleanup := setupPrefsTestConfig(t)
	defer cleanup()
	require.NoError(t, cfg.Profile.WriteConfigField("docs_prefs.lang", "ruby"))

	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, "# Page")
	}))
	defer server.Close()

	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	root := cmd.New().WithOptions(
		cmd.WithConfig(&cliconfig.Config{Color: "off", Profile: cfg.Profile}),
		cmd.WithClient(docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(new(bytes.Buffer))
	root.SetArgs([]string{"--non-interactive", "/page?lang=go"})

	require.NoError(t, root.ExecuteContext(context.Background()))
	assert.Contains(t, gotQuery, "lang=go")
	assert.NotContains(t, gotQuery, "lang=ruby")
}

func TestFetchPage_Authenticated(t *testing.T) {
	var gotPath, gotAuth, gotVersion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/docs/page", r.URL.Path)
		gotPath = r.URL.Query().Get("path")
		gotAuth = r.Header.Get("Authorization")
		gotVersion = r.Header.Get("Stripe-Version")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"content":"# Payments\n\nAccept payments with Stripe."}`)
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(
		docs.WithAPIBaseURL(server.URL),
		docs.WithAPIKey("sk_test_123"),
	)
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"--non-interactive", "/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
	assert.Equal(t, "/payments", gotPath)
	assert.Equal(t, "Bearer sk_test_123", gotAuth)
	assert.Equal(t, requests.StripeVersionHeaderValue, gotVersion)
}

func TestRootCommand_NoTUI_RendersOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept payments with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"--non-interactive", "/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
}
