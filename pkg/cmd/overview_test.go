package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	assert.Contains(t, out.String(), "Run `stripe docs /payments/payment-methods/dynamic-payment-methods` and follow the guide to make this change.")
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
		"We recommend this action to keep your integration healthy:\n1. Use dynamic payment methods. Run `stripe docs /payments/payment-methods/dynamic-payment-methods` and follow the guide to make this change.\n", out.String())
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

	require.ErrorContains(t, err, "integration health alerts aren't available in test mode")
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

	assert.JSONEq(t, `{
		"account": {"id": "acct_live", "mode": "live"},
		"default_api_version": "2024-06-20",
		"recommended_actions": [
			{"title": "Use dynamic payment methods", "guide": {"command": "stripe docs /payments/payment-methods/dynamic-payment-methods"}}
		]
	}`, out.String())
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

// stubOverviewDefaultAPIVersion also stubs the active account as a live OAuth
// account; sandbox tests override that with stubOverviewCredentials.
func stubOverviewDefaultAPIVersion(t *testing.T, version string) {
	t.Helper()
	orig := startOverviewDefaultAPIVersion
	startOverviewDefaultAPIVersion = func(context.Context, *config.Profile, string, stripe.Credentials) <-chan string {
		ch := make(chan string, 1)
		ch <- version
		return ch
	}
	t.Cleanup(func() { startOverviewDefaultAPIVersion = orig })
	stubOverviewCredentials(t, stripe.NewOAKCredentials("oak_123", "acct_live", true))
}

func stubOverviewCredentials(t *testing.T, creds stripe.Credentials) {
	t.Helper()
	orig := resolveOverviewCredentials
	resolveOverviewCredentials = func(*config.Profile, bool) (stripe.Credentials, error) {
		return creds, nil
	}
	t.Cleanup(func() { resolveOverviewCredentials = orig })
}

func TestOverviewCmd_RejectsNonPositiveLimit(t *testing.T) {
	for _, limit := range []string{"0", "-1"} {
		t.Run(limit, func(t *testing.T) {
			oc := newOverviewCmd()
			oc.cmd.SilenceUsage = true
			oc.cmd.SetOut(&bytes.Buffer{})
			oc.cmd.SetErr(&bytes.Buffer{})
			oc.cmd.SetArgs([]string{"--live", "--limit", limit})

			assert.EqualError(t, oc.cmd.Execute(), "--limit must be greater than 0")
		})
	}
}

func TestOverviewCmd_LimitsAlerts(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthPagedGateway(t, map[string]string{
		"": integHlthAlertsPage("", integHlthDPMAlertJSON("heins_1"), integHlthDPMAlertJSON("heins_2")),
	}, nil))
	stubOverviewDefaultAPIVersion(t, "")

	run := func(args ...string) []any {
		oc := newOverviewCmd()
		var out bytes.Buffer
		oc.cmd.SetOut(&out)
		oc.cmd.SetArgs(append([]string{"--live", "--format", "json"}, args...))
		require.NoError(t, oc.cmd.Execute())

		var got map[string]any
		require.NoError(t, json.Unmarshal(out.Bytes(), &got))
		return got["recommended_actions"].([]any)
	}

	assert.Len(t, run(), 2, "no --limit returns every alert")
	assert.Len(t, run("--limit", "1"), 1)
}

func TestRenderIntegHlthAlerts_PointsToStripeDocsForDocsStripeComDoctorDoc(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil)
	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)
	addIntegHlthAlertDoctorDocs(context.Background(), gatewayClient(server), resp.Alerts)

	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, resp.Alerts)

	assert.Equal(t, "We recommend this action to keep your integration healthy:\n1. Use dynamic payment methods. Run `stripe docs /payments/payment-methods/dynamic-payment-methods` and follow the guide to make this change.\n", buf.String())
}

func TestActionableIntegHlthAlerts_KeepsAlertsWithATitleAndStripeDocsFix(t *testing.T) {
	docsPage := "https://docs.stripe.com/payments/payment-methods/dynamic-payment-methods"
	otherPage := "https://stripe.com/guides/dynamic-payment-methods"
	summary := "Your integration pins payment_method_types"
	withTitle := &integHlthAlertInsight{RecommendedAction: &integHlthRecommendedAction{Title: "Use dynamic payment methods"}}
	alerts := []integHlthAlert{
		{HealthInsightID: "heins_title", HealthInsight: withTitle, DoctorDoc: &docsPage},
		{HealthInsightID: "heins_summary", HealthInsight: &integHlthAlertInsight{Summary: &summary}, DoctorDoc: &docsPage},
		{HealthInsightID: "heins_no_title", PredicateName: "LEGACY_PAYMENT_METHODS", DoctorDoc: &docsPage},
		{HealthInsightID: "heins_other_url", HealthInsight: withTitle, DoctorDoc: &otherPage},
		{HealthInsightID: "heins_no_doc", HealthInsight: withTitle},
	}

	var ids []string
	for _, a := range actionableIntegHlthAlerts(alerts) {
		ids = append(ids, a.HealthInsightID)
	}
	assert.Equal(t, []string{"heins_title", "heins_summary"}, ids)
	assert.NotNil(t, actionableIntegHlthAlerts(nil), "an empty list still encodes as [] in JSON")
}

func TestStripeDocsPath(t *testing.T) {
	for rawURL, want := range map[string]string{
		"https://docs.stripe.com":           "/",
		"https://docs.stripe.com/":          "/",
		"https://docs.stripe.com/payments":  "/payments",
		"https://docs.stripe.com.evil.test": "",
		"http://docs.stripe.com/payments":   "",
		"https://stripe.com/docs/payments":  "",
	} {
		path, ok := stripeDocsPath(rawURL)
		assert.Equal(t, want != "", ok, rawURL)
		assert.Equal(t, want, path, rawURL)
	}
}

func TestRenderIntegHlthAlerts_UsesActionTitleOrSummary(t *testing.T) {
	title := "Add the Stripe-Version header to your requests"
	summary := "Your requests don't set an API version."
	doc := "https://docs.stripe.com/upgrades"
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, []integHlthAlert{
		{HealthInsightID: "heins_123", DoctorDoc: &doc, HealthInsight: &integHlthAlertInsight{RecommendedAction: &integHlthRecommendedAction{Title: title}, Summary: &summary}},
		{HealthInsightID: "heins_456", DoctorDoc: &doc, HealthInsight: &integHlthAlertInsight{Summary: &summary}},
	})

	assert.Equal(t, "We recommend these 2 actions to keep your integration healthy:\n1. Add the Stripe-Version header to your requests. Run `stripe docs /upgrades` and follow the guide to make this change.\n"+
		"2. Your requests don't set an API version. Run `stripe docs /upgrades` and follow the guide to make this change.\n", buf.String())
}

func TestOverviewCmd_ShowsEmptyStateWhenNoAlertHasAFix(t *testing.T) {
	// The alert has no doctor topic, so there's no doc to point to.
	stubOverviewGateway(t, newIntegHlthPagedGateway(t, map[string]string{
		"": integHlthAlertsPage("", integHlthAlertJSON("heins_1", "LEGACY_PAYMENT_METHODS")),
	}, nil))
	stubOverviewDefaultAPIVersion(t, "")

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetArgs([]string{"--live"})
	require.NoError(t, oc.cmd.Execute())

	assert.Equal(t, "There are no recommended actions for your integration right now.\n", out.String())
}

func TestOverviewCmd_ClientSetupErrorShowsFriendlyMessage(t *testing.T) {
	stubOverviewDefaultAPIVersion(t, "")
	orig := newOverviewGraphQLClient
	newOverviewGraphQLClient = func(context.Context, graphql.Options) (genqlient.Client, error) {
		return nil, errors.New(`could not resolve the workspace for acct_123: Request failed, status=400, body={"error": {}}`)
	}
	t.Cleanup(func() { newOverviewGraphQLClient = orig })

	oc := newOverviewCmd()
	oc.cmd.SilenceUsage = true
	oc.cmd.SetOut(&bytes.Buffer{})
	oc.cmd.SetErr(&bytes.Buffer{})
	oc.cmd.SetArgs([]string{})

	assert.EqualError(t, oc.cmd.Execute(), overviewUnavailableMessage)
}

func TestRenderIntegHlthAlerts_Empty(t *testing.T) {
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, nil)

	assert.Equal(t, "There are no recommended actions for your integration right now.\n", buf.String())
}
