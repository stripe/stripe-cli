package stripe_test

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/config"
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

	oc := resource.NewOperationCmd(parentCmd, "foo", "/v1/bars/{id}", http.MethodGet, map[string]string{}, &config.Config{})
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
