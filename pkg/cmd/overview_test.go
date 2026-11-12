package cmd

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	genqlient "github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/requests"
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

	oc := newOverviewCmd()
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetArgs([]string{"--live"})
	require.NoError(t, oc.cmd.Execute())

	assert.True(t, gotOpts.Livemode)
	assert.Equal(t, requests.StripePreviewVersionHeaderValue, gotOpts.StripeVersion)
	assert.Contains(t, out.String(), "Run `stripe doctor dpm` for more information.")
}

func TestRenderIntegHlthAlerts_PointsToDoctorWhenTopicSet(t *testing.T) {
	server := newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil)
	resp, err := fetchIntegHlthAlerts(context.Background(), gatewayClient(server), 0)
	require.NoError(t, err)

	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, resp.Alerts)

	assert.Equal(t, "1 recommended action:\n1. Use dynamic payment methods. Run `stripe doctor dpm` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_PointsToHealthWithoutDoctorTopic(t *testing.T) {
	title := "Add the Stripe-Version header to your requests"
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, []integHlthAlert{
		{HealthInsightID: "heins_123", HealthInsight: &integHlthAlertInsight{RecommendedAction: &integHlthRecommendedAction{Title: title}}},
		{HealthInsightID: "heins_456", PredicateName: "SOME_PREDICATE"},
	})

	assert.Equal(t, "2 recommended actions:\n1. Add the Stripe-Version header to your requests. Run `stripe health heins_123` for more information.\n"+
		"2. SOME_PREDICATE. Run `stripe health heins_456` for more information.\n", buf.String())
}

func TestRenderIntegHlthAlerts_Empty(t *testing.T) {
	var buf bytes.Buffer
	renderIntegHlthAlerts(&buf, nil)

	assert.Contains(t, buf.String(), "No integration health alerts")
}
