package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	genqlient "github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestOverviewCmd_LiveUsesLiveModeAndPreviewVersion(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil)

	var gotOpts graphql.Options
	orig := newOverviewGraphQLClient
	newOverviewGraphQLClient = func(_ context.Context, opts graphql.Options) (genqlient.Client, error) {
		gotOpts = opts
		return gatewayClient(server), nil
	}
	t.Cleanup(func() { newOverviewGraphQLClient = orig })
	stubOverviewDefaultAPIVersion(t, "")

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetArgs([]string{"--live"})
	require.NoError(t, oc.cmd.Execute())

	assert.True(t, gotOpts.Livemode)
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, gotOpts.StripeVersion)
	assert.Contains(t, out.String(), "Run `stripe doctor dpm` for more information.")
	assert.NotContains(t, out.String(), "default API version")
}

func TestOverviewCmd_PrintsDefaultAPIVersion(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "2024-06-20")

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetArgs([]string{"--live"})
	require.NoError(t, oc.cmd.Execute())

	assert.Equal(t, "Your account's default API version is 2024-06-20.\n\n"+
		"We recommend this action to keep your integration healthy:\n1. Use dynamic payment methods. Run `stripe doctor dpm` for more information.\n", out.String())
}

func TestOverviewCmd_PrintsDefaultAPIVersionWhenAlertsFail(t *testing.T) {
	body := `{"errors":[{"message":"not in testmode","extensions":{"code":"v2_api_not_supported_in_testmode"}}]}`
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, body, nil))
	stubOverviewDefaultAPIVersion(t, "2024-06-20")

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SilenceUsage = true
	oc.cmd.SetArgs([]string{"--live"})
	err := oc.cmd.Execute()

	require.ErrorContains(t, err, "stripe switch --live")
	assert.Equal(t, "Your account's default API version is 2024-06-20.\n\n", out.String())
}

func TestOverviewCmd_JSONIncludesDefaultAPIVersion(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "2024-06-20")

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetArgs([]string{"--live", "--format", "json"})
	require.NoError(t, oc.cmd.Execute())

	var got map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &got))
	assert.Equal(t, "2024-06-20", got["default_api_version"])
	assert.Len(t, got["alerts"], 1)
}

func TestDefaultAPIVersion_SkipsAPIKeys(t *testing.T) {
	orig := getAccountFacts
	getAccountFacts = func(context.Context, string, *config.Profile, stripe.Credentials, bool) (requests.AccountFacts, error) {
		t.Fatal("account facts should not be requested with an API key")
		return requests.AccountFacts{}, nil
	}
	t.Cleanup(func() { getAccountFacts = orig })

	assert.Empty(t, defaultAPIVersion(context.Background(), "", nil, stripe.Credentials{Token: "sk_test_123"}, false))
}

func TestDefaultAPIVersion_UsesOAuthCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/stripecli/account_facts", r.URL.Path)
		assert.Equal(t, "acct_123", r.Header.Get("Stripe-Context"))
		_, _ = w.Write([]byte(`{"api_versions":["2024-06-20"],"default_api_version":"2024-06-20"}`))
	}))
	t.Cleanup(server.Close)

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	assert.Equal(t, "2024-06-20", defaultAPIVersion(context.Background(), server.URL, nil, creds, true))
}

func TestDefaultAPIVersion_SwallowsErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"Unrecognized request URL"}}`))
	}))
	t.Cleanup(server.Close)

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_123", true)
	assert.Empty(t, defaultAPIVersion(context.Background(), server.URL, nil, creds, true))
}

func stubOverviewGateway(t *testing.T, server *httptest.Server) {
	t.Helper()
	orig := newOverviewGraphQLClient
	newOverviewGraphQLClient = func(context.Context, graphql.Options) (genqlient.Client, error) {
		return gatewayClient(server), nil
	}
	t.Cleanup(func() { newOverviewGraphQLClient = orig })
}

func stubOverviewDefaultAPIVersion(t *testing.T, version string) {
	t.Helper()
	orig := startOverviewDefaultAPIVersion
	startOverviewDefaultAPIVersion = func(context.Context, *config.Profile, string, bool) <-chan string {
		ch := make(chan string, 1)
		ch <- version
		return ch
	}
	t.Cleanup(func() { startOverviewDefaultAPIVersion = orig })
}

func TestRenderIntegHlthAlerts_PointsToDoctorWhenTopicSet(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil)
	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, resp.Alerts)

	assert.Equal(t, "We recommend this action to keep your integration healthy:\n1. Use dynamic payment methods. Run `stripe doctor dpm` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_PointsToHealthWithoutDoctorTopic(t *testing.T) {
	title := "Add the Stripe-Version header to your requests"
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, []integHlthAlert{
		{HealthInsightID: "heins_123", HealthInsight: &integHlthAlertInsight{RecommendedAction: &integHlthRecommendedAction{Title: title}}},
		{HealthInsightID: "heins_456", PredicateName: "SOME_PREDICATE"},
	})

	assert.Equal(t, "We recommend these 2 actions to keep your integration healthy:\n1. Add the Stripe-Version header to your requests. Run `stripe health heins_123` for more information.\n"+
		"2. SOME_PREDICATE. Run `stripe health heins_456` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_Empty(t *testing.T) {
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, nil)

	assert.Equal(t, "There are no recommended actions for your integration right now.\n", buf.String())
}
