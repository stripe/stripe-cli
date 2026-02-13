package stripe_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/spec"
	"github.com/stripe/stripe-cli/pkg/stripe"
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

	oc := resource.NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodGet, map[string]string{}, map[string][]spec.StripeEnumValue{}, &config.Config{}, false)
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
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "livemode=false")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
		require.Contains(t, bodyString, "request_id=req_zzz")
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
		Merchant:          "acct_1234",
		GeneratedResource: false,
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
		require.Contains(t, bodyString, "invocation_id=123456")
		require.Contains(t, bodyString, "merchant=acct_1234")
		require.Contains(t, bodyString, "os=darwin")
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
		Merchant:          "acct_1234",
		GeneratedResource: false,
	}
	processCtx := stripe.WithEventMetadata(context.Background(), telemetryMetadata)
	analyticsClient := stripe.AnalyticsTelemetryClient{BaseURL: baseURL, HTTPClient: &http.Client{}}
	analyticsClient.SendEvent(processCtx, "foo", "bar")
}

func TestSendEvent_WithAIAgent(t *testing.T) {
	allEnvVars := []string{
		"ANTIGRAVITY_CLI_ALIAS",
		"CLAUDECODE",
		"CLINE_ACTIVE",
		"CODEX_SANDBOX",
		"CURSOR_AGENT",
		"GEMINI_CLI",
		"OPENCODE",
	}

	// Save all env vars
	savedEnvs := make(map[string]string)
	for _, envVar := range allEnvVars {
		savedEnvs[envVar] = os.Getenv(envVar)
	}

	// Restore all env vars after test
	defer func() {
		for envVar, val := range savedEnvs {
			if val != "" {
				os.Setenv(envVar, val)
			} else {
				os.Unsetenv(envVar)
			}
		}
	}()

	// Clear all env vars first
	for _, envVar := range allEnvVars {
		os.Unsetenv(envVar)
	}

	// Set only CURSOR_AGENT
	os.Setenv("CURSOR_AGENT", "1")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		bodyString := string(body)
		require.Contains(t, bodyString, "ai_agent=cursor")
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	// Create metadata using NewEventMetadata which will detect the env var
	telemetryMetadata := stripe.NewEventMetadata()
	telemetryMetadata.InvocationID = "123456"
	telemetryMetadata.UserAgent = "Unit Test"
	telemetryMetadata.CLIVersion = "master"
	telemetryMetadata.OS = "darwin"
	telemetryMetadata.CommandPath = "stripe test"
	telemetryMetadata.Merchant = "acct_1234"

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
func TestNewEventMetadata_DetectsAIAgent(t *testing.T) {
	// Save original env state
	originalEnv := os.Getenv("CLAUDECODE")
	defer func() {
		if originalEnv != "" {
			os.Setenv("CLAUDECODE", originalEnv)
		} else {
			os.Unsetenv("CLAUDECODE")
		}
	}()

	// Test with CLAUDECODE set
	os.Setenv("CLAUDECODE", "1")
	tel := stripe.NewEventMetadata()
	require.Equal(t, "claude_code", tel.AIAgent)

	// Test with no AI agent env vars
	os.Unsetenv("CLAUDECODE")
	tel = stripe.NewEventMetadata()
	require.Equal(t, "", tel.AIAgent)
}

func TestAIAgentDetection_AllAgents(t *testing.T) {
	allEnvVars := []string{
		"ANTIGRAVITY_CLI_ALIAS",
		"CLAUDECODE",
		"CLINE_ACTIVE",
		"CODEX_SANDBOX",
		"CURSOR_AGENT",
		"GEMINI_CLI",
		"OPENCODE",
	}

	tests := []struct {
		name     string
		envVar   string
		expected string
	}{
		{"Antigravity", "ANTIGRAVITY_CLI_ALIAS", "antigravity"},
		{"Claude Code", "CLAUDECODE", "claude_code"},
		{"Cline", "CLINE_ACTIVE", "cline"},
		{"Codex CLI", "CODEX_SANDBOX", "codex_cli"},
		{"Cursor", "CURSOR_AGENT", "cursor"},
		{"Gemini CLI", "GEMINI_CLI", "gemini_cli"},
		{"Open Code", "OPENCODE", "open_code"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save all env vars
			savedEnvs := make(map[string]string)
			for _, envVar := range allEnvVars {
				savedEnvs[envVar] = os.Getenv(envVar)
			}

			// Restore all env vars after test
			defer func() {
				for envVar, val := range savedEnvs {
					if val != "" {
						os.Setenv(envVar, val)
					} else {
						os.Unsetenv(envVar)
					}
				}
			}()

			// Clear all env vars first
			for _, envVar := range allEnvVars {
				os.Unsetenv(envVar)
			}

			// Set only the env var being tested
			os.Setenv(tt.envVar, "true")
			tel := stripe.NewEventMetadata()
			require.Equal(t, tt.expected, tel.AIAgent)
		})
	}
}

func TestAIAgentDetection_Priority(t *testing.T) {
	// Test that the first matching env var wins
	// Save original env state
	originalAntigravity := os.Getenv("ANTIGRAVITY_CLI_ALIAS")
	originalCursor := os.Getenv("CURSOR_AGENT")
	defer func() {
		if originalAntigravity != "" {
			os.Setenv("ANTIGRAVITY_CLI_ALIAS", originalAntigravity)
		} else {
			os.Unsetenv("ANTIGRAVITY_CLI_ALIAS")
		}
		if originalCursor != "" {
			os.Setenv("CURSOR_AGENT", originalCursor)
		} else {
			os.Unsetenv("CURSOR_AGENT")
		}
	}()

	// Set multiple env vars - should return the first one (antigravity)
	os.Setenv("ANTIGRAVITY_CLI_ALIAS", "1")
	os.Setenv("CURSOR_AGENT", "1")
	tel := stripe.NewEventMetadata()
	require.Equal(t, "antigravity", tel.AIAgent)

	// Clean up
	os.Unsetenv("ANTIGRAVITY_CLI_ALIAS")
	os.Unsetenv("CURSOR_AGENT")
}
