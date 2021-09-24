package stripe

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Context Tests
func TestSetCobraCommandContext(t *testing.T) {
	tel := InitContext()
	cmd := &cobra.Command{
		Use: "foo",
	}
	tel.SetCobraCommandContext(cmd)
	require.Equal(t, "foo", tel.CommandPath)
}

func TestSetMerchant(t *testing.T) {
	tel := InitContext()
	merchant := "acct_zzzzzz"
	tel.SetMerchant(merchant)
	require.Equal(t, merchant, tel.Merchant)
}

// AnalyticsClient Tests
func TestSendAPIRequestEvent(t *testing.T) {
	os.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "0")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
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

	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryContext := &CLIAnalyticsEventContext{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		Merchant:          "acct_1234",
		GeneratedResource: false,
	}
	processCtx := context.WithValue(context.Background(), TelemetryContextKey{}, telemetryContext)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, WG: &sync.WaitGroup{}, HttpClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(processCtx, "req_zzz", false)
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
}

func TestSkipsSendAPIRequestWhenUserOptsOutOfTelemetry(t *testing.T) {
	os.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "1")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryContext := InitContext()
	processCtx := context.WithValue(context.Background(), TelemetryContextKey{}, telemetryContext)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, WG: &sync.WaitGroup{}, HttpClient: &http.Client{}}
	resp, err := analyticsClient.SendAPIRequestEvent(processCtx, "req_zzz", false)
	require.NoError(t, err)
	require.Nil(t, resp)
}

func TestSendEvent(t *testing.T) {
	os.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "0")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
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

	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryContext := &CLIAnalyticsEventContext{
		InvocationID:      "123456",
		UserAgent:         "Unit Test",
		CLIVersion:        "master",
		OS:                "darwin",
		CommandPath:       "stripe test",
		Merchant:          "acct_1234",
		GeneratedResource: false,
	}
	processCtx := context.WithValue(context.Background(), TelemetryContextKey{}, telemetryContext)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, WG: &sync.WaitGroup{}, HttpClient: &http.Client{}}
	resp, err := analyticsClient.SendEvent(processCtx, "foo", "bar")
	require.NoError(t, err)
	require.NotNil(t, resp)
	resp.Body.Close()
}

func TestSkipsSendEventWhenUserOptsOutOfTelemetry(t *testing.T) {
	os.Setenv("STRIPE_CLI_TELEMETRY_OPTOUT", "1")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// do nothing
	}))
	defer ts.Close()
	baseURL, _ := url.Parse(ts.URL)

	telemetryContext := InitContext()
	processCtx := context.WithValue(context.Background(), TelemetryContextKey{}, telemetryContext)
	analyticsClient := AnalyticsTelemetryClient{BaseURL: baseURL, WG: &sync.WaitGroup{}, HttpClient: &http.Client{}}
	resp, err := analyticsClient.SendEvent(processCtx, "foo", "bar")
	require.NoError(t, err)
	require.Nil(t, resp)
}

// Utility function
func TestTelemetryOptedOut(t *testing.T) {
	require.False(t, telemetryOptedOut(""))
	require.False(t, telemetryOptedOut("0"))
	require.False(t, telemetryOptedOut("false"))
	require.False(t, telemetryOptedOut("False"))
	require.False(t, telemetryOptedOut("FALSE"))
	require.True(t, telemetryOptedOut("1"))
	require.True(t, telemetryOptedOut("true"))
	require.True(t, telemetryOptedOut("True"))
	require.True(t, telemetryOptedOut("TRUE"))
}
