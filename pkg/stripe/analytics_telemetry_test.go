package stripe_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/useragent"
)

// Context Tests
func TestEventMetadataWithGet(t *testing.T) {
	ctx := context.Background()
	event := &stripe.CLIAnalyticsEventMetadata{
		InvocationID: "hello",
		UserAgent:    "uesr",
		CLIVersion:   "1.0",
		OS:           "os",
	}
	newCtx := stripe.WithEventMetadata(ctx, event)

	require.Equal(t, stripe.GetEventMetadata(newCtx), event)
}

func TestGetEventMetadata_DoesNotExistInCtx(t *testing.T) {
	ctx := context.Background()
	require.Nil(t, stripe.GetEventMetadata(ctx))
}

func TestTelemetryClientWithGet(t *testing.T) {
	ctx := context.Background()
	url, _ := url.Parse("http://hello.com")
	telemetryClient := &stripe.AnalyticsTelemetryClient{
		BaseURL:    url,
		HTTPClient: &http.Client{},
	}
	newCtx := stripe.WithTelemetryClient(ctx, telemetryClient)

	require.Equal(t, stripe.GetTelemetryClient(newCtx), telemetryClient)
}

func TestGetTelemetryClient_DoesNotExistInCtx(t *testing.T) {
	ctx := context.Background()
	require.Nil(t, stripe.GetTelemetryClient(ctx))
}

func TestSetCobraCommandContext(t *testing.T) {
	tel := stripe.NewEventMetadata()
	cmd := &cobra.Command{
		Use: "foo",
	}
	tel.SetCobraCommandContext(cmd)
	require.Equal(t, "foo", tel.CommandPath)
	require.False(t, tel.GeneratedResource)
}

func TestSetCobraCommandContext_SetsGeneratedResourceForGeneratedCommands(t *testing.T) {
	parentCmd := &cobra.Command{Annotations: make(map[string]string)}

	oc := resource.NewOperationCmd(parentCmd, &resource.OperationSpec{
		Name:   "foo",
		Path:   "/v1/bars/{id}",
		Method: http.MethodGet,
	}, &config.Config{})
	tel := stripe.NewEventMetadata()
	tel.SetCobraCommandContext(oc.Cmd)
	require.True(t, tel.GeneratedResource)
}

func TestSetMerchant(t *testing.T) {
	tel := stripe.NewEventMetadata()
	merchant := "acct_zzzzzz"
	tel.SetMerchant(merchant)
	require.Equal(t, merchant, tel.Merchant)
}

func TestSetPluginName(t *testing.T) {
	tel := stripe.NewEventMetadata()
	tel.SetPluginName("apps")
	require.Equal(t, "apps", tel.PluginName)
}

// AnalyticsClient Tests
func TestSendAPIRequestEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "cli_version=master")
		require.Contains(t, bodyString, "client_id=stripe-cli")
		require.Contains(t, bodyString, "command_path=stripe+test")
		require.Contains(t, bodyString, "event_name=API+Request")
		require.Contains(t, bodyString, "generated_resource=false")
		require.Contains(t, bodyString, "in_screen=true")
		require.Contains(t, bodyString, "in_tmux=true")
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "livemode=false")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
		require.Contains(t, bodyString, "plugin_name=apps")
		require.Contains(t, bodyString, "request_id=req_zzz")
		require.Contains(t, bodyString, "terminal_program=iTerm.app")
		require.Contains(t, bodyString, "user_agent=Unit+Test")
		// ai_agent should not be present when empty (omitempty)
		require.NotContains(t, bodyString, "ai_agent")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &stripe.CLIAnalyticsEventMetadata{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		PluginName:        "apps",
		Merchant:          "acct_1234",
		GeneratedResource: false,
		InTmux:            true,
		InScreen:          true,
		TerminalProgram:   "iTerm.app",
	}
	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(processCtx, "req_zzz", false)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
}

func TestSkipsSendAPIRequestEventWhenMetadataIsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(context.Background(), "req_zzz", false)
	require.NoError(t, err)
	require.Nil(t, resp)

	// We shouldn't get here but the linter is unhappy
	if resp != nil {
		resp.Body.Close()
	}
}

func TestSendEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "cli_version=master")
		require.Contains(t, bodyString, "client_id=stripe-cli")
		require.Contains(t, bodyString, "command_path=stripe+test")
		require.Contains(t, bodyString, "event_name=foo")
		require.Contains(t, bodyString, "event_value=bar")
		require.Contains(t, bodyString, "generated_resource=false")
		require.Contains(t, bodyString, "in_screen=true")
		require.Contains(t, bodyString, "in_tmux=true")
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
		require.Contains(t, bodyString, "plugin_name=apps")
		require.Contains(t, bodyString, "terminal_program=iTerm.app")
		require.Contains(t, bodyString, "user_agent=Unit+Test")
		// ai_agent should not be present when empty (omitempty)
		require.NotContains(t, bodyString, "ai_agent")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &stripe.CLIAnalyticsEventMetadata{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		PluginName:        "apps",
		Merchant:          "acct_1234",
		GeneratedResource: false,
		InTmux:            true,
		InScreen:          true,
		TerminalProgram:   "iTerm.app",
	}
	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestHTTPOutcomeForStatusCode(t *testing.T) {
	for _, test := range []struct {
		status int
		want   string
	}{
		{status: -1, want: ""},
		{status: 0, want: ""},
		{status: 100, want: "other_non_200"},
		{status: 199, want: "other_non_200"},
		{status: 200, want: "2xx"},
		{status: 201, want: "2xx"},
		{status: 204, want: "2xx"},
		{status: 299, want: "2xx"},
		{status: 300, want: "other_non_200"},
		{status: 302, want: "other_non_200"},
		{status: 399, want: "other_non_200"},
		{status: 400, want: "other_4xx"},
		{status: 401, want: "other_4xx"},
		{status: 403, want: "other_4xx"},
		{status: 404, want: "not_found"},
		{status: 429, want: "other_4xx"},
		{status: 499, want: "other_4xx"},
		{status: 500, want: "5xx"},
		{status: 599, want: "5xx"},
		{status: 600, want: "other_non_200"},
	} {
		t.Run(fmt.Sprint(test.status), func(t *testing.T) {
			require.Equal(t, test.want, stripe.HTTPOutcomeForStatusCode(test.status))
		})
	}
}

func TestSendAPIRequestEvent_HTTPOutcome(t *testing.T) {
	for _, status := range []int{0, 200, 201, 204, 302, 400, 401, 403, 404, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			forms := make(chan url.Values, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				forms <- r.PostForm
			}))
			defer server.Close()
			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			client := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: server.Client()}
			outcome := stripe.HTTPOutcomeForStatusCode(status)
			metadata := &stripe.CLIAnalyticsEventMetadata{CommandPath: "stripe get", HTTPOutcome: outcome}
			ctx := stripe.WithEventMetadata(context.Background(), metadata)

			resp, err := client.SendAPIRequestEvent(ctx, "", false)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())

			form := <-forms
			require.Equal(t, "API Request", form.Get("event_name"))
			require.Equal(t, outcome, form.Get("http_outcome"))
			require.Equal(t, outcome != "", form.Has("http_outcome"))
			require.False(t, form.Has("http_status_code"))
		})
	}
}

func TestSendEvent_HTTPOutcome(t *testing.T) {
	for _, outcome := range []string{"", "not_found", "other_4xx", "5xx", "other_non_200"} {
		t.Run("outcome="+outcome, func(t *testing.T) {
			forms := make(chan url.Values, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if err := r.ParseForm(); err != nil {
					t.Error(err)
				}
				forms <- r.PostForm
			}))
			defer server.Close()
			baseURL, err := url.Parse(server.URL)
			require.NoError(t, err)
			client := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: server.Client()}
			metadata := &stripe.CLIAnalyticsEventMetadata{CommandPath: "get", HTTPOutcome: outcome}
			ctx := stripe.WithEventMetadata(context.Background(), metadata)

			client.SendEvent(ctx, "CLI Error", "api")

			form := <-forms
			require.Equal(t, "CLI Error", form.Get("event_name"))
			require.Equal(t, "api", form.Get("event_value"))
			require.Equal(t, outcome, form.Get("http_outcome"))
			require.Equal(t, outcome != "", form.Has("http_outcome"))
			require.False(t, form.Has("http_status_code"))
		})
	}
}

func TestSendEvent_WithAIAgent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "ai_agent=cursor")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	// Create metadata and manually set AIAgent field
	telemetryMetadata := &stripe.CLIAnalyticsEventMetadata{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		Merchant:          "acct_1234",
		GeneratedResource: false,
		AIAgent:           "cursor",
	}

	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestSendEvent_WithAgentHost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "ai_agent=claude_code")
		require.Contains(t, bodyString, "agent_version=2.1.222")
		require.Contains(t, bodyString, "agent_host_kind=desktop")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &stripe.CLIAnalyticsEventMetadata{
		InvocationID:  "123456",
		UserAgent:     "Unit Test",
		CLIVersion:    "master",
		OS:            "darwin",
		CommandPath:   "stripe test",
		Merchant:      "acct_1234",
		AIAgent:       "claude_code",
		AgentHostKind: "desktop",
		AgentVersion:  "2.1.222",
	}

	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestSendEvent_OmitsUnsetAgentFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.NotContains(t, bodyString, "agent_version")
		require.NotContains(t, bodyString, "agent_host_kind")
		require.NotContains(t, bodyString, "agent_host_raw")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryMetadata := &stripe.CLIAnalyticsEventMetadata{
		InvocationID: "123456",
		UserAgent:    "Unit Test",
		CLIVersion:   "master",
		OS:           "linux",
		CommandPath:  "stripe test",
		Merchant:     "acct_1234",
	}

	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestSkipsSendEventWhenMetadataIsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Fail(t, "Did not expect to reach sendData")
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(context.Background(), "foo", "bar")
}

// Utility function
func TestTelemetryOptedOut(t *testing.T) {
	require.False(t, stripe.TelemetryOptedOut(""))
	require.False(t, stripe.TelemetryOptedOut("0"))
	require.False(t, stripe.TelemetryOptedOut("false"))
	require.False(t, stripe.TelemetryOptedOut("False"))
	require.False(t, stripe.TelemetryOptedOut("FALSE"))
	require.True(t, stripe.TelemetryOptedOut("1"))
	require.True(t, stripe.TelemetryOptedOut("true"))
	require.True(t, stripe.TelemetryOptedOut("True"))
	require.True(t, stripe.TelemetryOptedOut("TRUE"))
}

// AI Agent Detection Tests
func TestDetectAIAgent_WithClaudeCode(t *testing.T) {
	// Mock env getter that returns CLAUDECODE
	getEnv := func(key string) string {
		if key == "CLAUDECODE" {
			return "1"
		}
		return ""
	}
	result := useragent.DetectAIAgent(getEnv)
	require.Equal(t, "claude_code", result)
}

func TestDetectAIAgent_NoAgentDetected(t *testing.T) {
	// Mock env getter that returns empty strings
	getEnv := func(key string) string {
		return ""
	}
	result := useragent.DetectAIAgent(getEnv)
	require.Equal(t, "", result)
}

func TestAIAgentDetection_AllAgents(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected string
	}{
		{"Antigravity", "ANTIGRAVITY_CLI_ALIAS", "true", "antigravity"},
		{"Claude Code", "CLAUDECODE", "true", "claude_code"},
		{"Cline", "CLINE_ACTIVE", "true", "cline"},
		{"Codex CLI sandbox", "CODEX_SANDBOX", "true", "codex_cli"},
		{"Codex CLI thread", "CODEX_THREAD_ID", "thread-id", "codex_cli"},
		{"Codex CLI sandbox network", "CODEX_SANDBOX_NETWORK_DISABLED", "true", "codex_cli"},
		{"Codex CLI CI", "CODEX_CI", "true", "codex_cli"},
		{"Cursor", "CURSOR_AGENT", "true", "cursor"},
		{"Gemini CLI", "GEMINI_CLI", "true", "gemini_cli"},
		{"Open Code", "OPENCODE", "true", "open_code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock env getter that returns only the specific env var being tested
			getEnv := func(key string) string {
				if key == tt.envVar {
					return tt.envValue
				}
				return ""
			}
			result := useragent.DetectAIAgent(getEnv)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestAIAgentDetection_Priority(t *testing.T) {
	// Test that the first matching env var wins (antigravity comes before cursor)
	getEnv := func(key string) string {
		if key == "ANTIGRAVITY_CLI_ALIAS" || key == "CURSOR_AGENT" {
			return "1"
		}
		return ""
	}
	result := useragent.DetectAIAgent(getEnv)
	require.Equal(t, "antigravity", result)
}

func clearAgentEnv(t *testing.T) {
	for _, key := range []string{
		"ANTIGRAVITY_CLI_ALIAS", "CLAUDECODE", "CLINE_ACTIVE", "CODEX_INTERNAL_ORIGINATOR_OVERRIDE",
		"CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI",
		"CURSOR_AGENT", "GEMINI_CLI", "OPENCODE", "OPENCLAW_SHELL",
		"AI_AGENT", "AGENT", "CLAUDE_CODE_ENTRYPOINT", "__CFBundleIdentifier",
	} {
		t.Setenv(key, "")
	}
}

func TestNewEventMetadata_FromObservedDesktopSession(t *testing.T) {
	// Values observed in a real Claude Desktop session; see TestObservedAgentSessions
	// in pkg/useragent for the full set and for Codex Desktop.
	clearAgentEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("AI_AGENT", "claude-code_2-1-227_agent")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "claude-desktop")

	tel := stripe.NewEventMetadata()
	require.Equal(t, "claude_code", tel.AIAgent)
	require.Equal(t, "desktop", tel.AgentHostKind)
	require.Equal(t, "2.1.227", tel.AgentVersion)
	require.Equal(t, "claude-desktop", tel.AgentHostRaw)
}

func TestNewEventMetadata_ReportsRawHostWhenUncategorized(t *testing.T) {
	// The debugging case: a host neither vendor had shipped when this CLI was released.
	// The category cannot identify it; the raw value can.
	clearAgentEnv(t)
	t.Setenv("CLAUDECODE", "1")
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "some-future-surface")

	tel := stripe.NewEventMetadata()
	require.Equal(t, "other", tel.AgentHostKind)
	require.Equal(t, "some-future-surface", tel.AgentHostRaw)
}

func TestNewEventMetadata_InfersCodexTerminalFromAbsentOriginator(t *testing.T) {
	// Codex sets its originator override only from non-default surfaces, so a terminal
	// session reports no host at all. Without the inference this is indistinguishable
	// from a CLI too old to report a host, and terminal Codex is invisible.
	clearAgentEnv(t)
	t.Setenv("CODEX_SANDBOX", "1")

	tel := stripe.NewEventMetadata()
	require.Equal(t, "codex_cli", tel.AIAgent)
	require.Equal(t, "terminal", tel.AgentHostKind)
	require.Equal(t, "codex-cli", tel.AgentHostRaw)
}

func TestNewEventMetadata_InfersAgentFromHostAlone(t *testing.T) {
	// A Claude Code entrypoint with no CLAUDECODE: hosted and SDK surfaces do not always
	// set it. The entrypoint names the agent, so the two fields stay consistent.
	clearAgentEnv(t)
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "remote-trigger")

	tel := stripe.NewEventMetadata()
	require.Equal(t, "claude_code", tel.AIAgent)
	require.Equal(t, "remote", tel.AgentHostKind)
	require.Equal(t, "remote-trigger", tel.AgentHostRaw)
}

func TestNewEventMetadata_CategorizesCodexVSCode(t *testing.T) {
	clearAgentEnv(t)
	t.Setenv("CODEX_CI", "1")
	t.Setenv("CODEX_INTERNAL_ORIGINATOR_OVERRIDE", "codex_vscode")

	tel := stripe.NewEventMetadata()
	require.Equal(t, "codex_cli", tel.AIAgent)
	require.Equal(t, "ide", tel.AgentHostKind)
	require.Equal(t, "codex-vscode", tel.AgentHostRaw)
}

type recordingRoundTripper struct {
	mu   sync.Mutex
	urls []string
}

func (r *recordingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.urls = append(r.urls, req.URL.String())
	r.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("")),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func TestConcurrentSendsWithDefaultBaseURL(t *testing.T) {
	rt := &recordingRoundTripper{}
	client := &stripe.AnalyticsTelemetryClient{HTTPClient: &http.Client{Transport: rt}}
	ctx := stripe.WithEventMetadata(context.Background(), stripe.NewEventMetadata())

	const n = 8
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			client.SendEvent(ctx, "Command Invoked", "")
		}()
		go func() {
			defer wg.Done()
			resp, err := client.SendAPIRequestEvent(ctx, "req_123", false)
			if err == nil && resp != nil {
				resp.Body.Close()
			}
		}()
	}
	wg.Wait()

	require.Len(t, rt.urls, 2*n)
	for _, u := range rt.urls {
		require.Equal(t, stripe.DefaultTelemetryEndpoint, u)
	}
	require.Nil(t, client.BaseURL)
}
