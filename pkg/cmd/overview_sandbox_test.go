package cmd

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	genqlient "github.com/Khan/genqlient/graphql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/graphql"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const (
	sandboxForceLiveFooter = "\nYou're currently in a sandbox, so these insights are only applicable to your sandbox.\n" +
		"Run `stripe overview --force-live` to see insights for your live account.\n"
	sandboxLoginFooter = "\nYou're currently in a sandbox, so these insights are only applicable to your sandbox.\n" +
		"Run `stripe login` to authorize your live account.\n" +
		"Then run `stripe overview --force-live` to see insights for your live account.\n"
	overviewDPMAction = "We recommend this action to keep your integration healthy:\n" +
		"1. Use dynamic payment methods. Run `stripe docs /payments/payment-methods/dynamic-payment-methods` and follow the guide to make this change.\n"
)

var sandboxCreds = stripe.NewOAKCredentials("oak_123", "acct_sandbox", false)

func stubOverviewSandboxParent(t *testing.T, parentID string, ok bool) *[]stripe.Credentials {
	t.Helper()
	var calls []stripe.Credentials
	orig := lookupOverviewSandboxParent
	lookupOverviewSandboxParent = func(_ context.Context, _ string, _ *config.Profile, creds stripe.Credentials) (string, bool) {
		calls = append(calls, creds)
		return parentID, ok
	}
	t.Cleanup(func() { lookupOverviewSandboxParent = orig })
	return &calls
}

func stubOverviewGatewayCapturingOptions(t *testing.T, server *httptest.Server) *graphql.Options {
	t.Helper()
	var got graphql.Options
	orig := newOverviewGraphQLClient
	newOverviewGraphQLClient = func(_ context.Context, opts graphql.Options) (genqlient.Client, error) {
		got = opts
		return gatewayClient(server), nil
	}
	t.Cleanup(func() { newOverviewGraphQLClient = orig })
	return &got
}

func runOverview(t *testing.T, args ...string) (string, error) {
	t.Helper()
	oc := newOverviewCmd()
	oc.cmd.SilenceUsage = true
	oc.cmd.SilenceErrors = true
	var out bytes.Buffer
	oc.cmd.SetOut(&out)
	oc.cmd.SetErr(&bytes.Buffer{})
	oc.cmd.SetArgs(args)
	err := oc.cmd.Execute()
	return out.String(), err
}

func TestOverviewCmd_SandboxWithAuthorizedLiveAccountPointsToForceLive(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	calls := stubOverviewSandboxParent(t, "acct_live", true)

	out, err := runOverview(t)
	require.NoError(t, err)

	assert.Equal(t, overviewDPMAction+sandboxForceLiveFooter, out)
	require.Len(t, *calls, 1)
	assert.Equal(t, "acct_sandbox", (*calls)[0].OAKContext)
}

func TestOverviewCmd_SandboxWithoutAuthorizedLiveAccountPointsToLogin(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "", true)

	out, err := runOverview(t)
	require.NoError(t, err)

	assert.Equal(t, overviewDPMAction+sandboxLoginFooter, out)
}

func TestOverviewCmd_SandboxParentLookupFailurePointsToForceLive(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "", false)

	out, err := runOverview(t)
	require.NoError(t, err)

	assert.Equal(t, overviewDPMAction+sandboxForceLiveFooter, out, "an unknown parent shouldn't tell an authorized user to log in")
}

func TestOverviewCmd_SandboxFooterFollowsEmptyStateAndErrors(t *testing.T) {
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "acct_live", true)

	t.Run("empty state", func(t *testing.T) {
		stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, `{"data": {"v2ListIntegHlthAlerts": {"alerts": []}}}`, nil))

		out, err := runOverview(t)
		require.NoError(t, err)
		assert.Equal(t, "There are no recommended actions for your integration right now.\n"+sandboxForceLiveFooter, out)
	})

	t.Run("alerts error", func(t *testing.T) {
		stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusInternalServerError, `{"errors": [{"message": "gateway unavailable"}]}`, nil))

		out, err := runOverview(t)
		require.Error(t, err)
		assert.Equal(t, sandboxForceLiveFooter, out)
	})
}

func TestOverviewCmd_LiveAccountHasNoSandboxFooter(t *testing.T) {
	stubOverviewGateway(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	calls := stubOverviewSandboxParent(t, "", true)

	out, err := runOverview(t)
	require.NoError(t, err)

	assert.Equal(t, overviewDPMAction, out)
	assert.Empty(t, *calls, "live accounts don't look up a parent")
}

func TestOverviewCmd_ForceLiveReadsTheLiveAccount(t *testing.T) {
	opts := stubOverviewGatewayCapturingOptions(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "acct_live", true)

	var versionCreds stripe.Credentials
	origVersion := startOverviewDefaultAPIVersion
	startOverviewDefaultAPIVersion = func(_ context.Context, _ *config.Profile, _ string, creds stripe.Credentials) <-chan string {
		versionCreds = creds
		ch := make(chan string, 1)
		ch <- "2024-06-20"
		return ch
	}
	t.Cleanup(func() { startOverviewDefaultAPIVersion = origVersion })

	out, err := runOverview(t, "--force-live")
	require.NoError(t, err)

	assert.Equal(t, "Showing insights for your live account (acct_live).\n\n"+
		"Your account's default API version is 2024-06-20.\n\n"+overviewDPMAction, out)

	want := stripe.NewOAKCredentials("oak_123", "acct_live", true)
	require.NotNil(t, opts.Credentials)
	assert.Equal(t, want, *opts.Credentials, "alerts are read with the live account's credentials")
	assert.True(t, opts.Livemode)
	assert.Equal(t, want, versionCreds, "the default API version is the live account's")
}

func TestOverviewCmd_ForceLiveWithoutAuthorizedLiveAccount(t *testing.T) {
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "", true)

	_, err := runOverview(t, "--force-live")

	require.EqualError(t, err, "Your live account isn't authorized. Run `stripe login` to authorize it, then run `stripe overview --force-live`.")
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	assert.Equal(t, errorcategory.UserInput, category)
}

func TestOverviewCmd_ForceLiveParentLookupFailure(t *testing.T) {
	stubOverviewDefaultAPIVersion(t, "")
	stubOverviewCredentials(t, sandboxCreds)
	stubOverviewSandboxParent(t, "", false)

	_, err := runOverview(t, "--force-live")

	require.EqualError(t, err, "could not find the live account for your sandbox; try again")
}

func TestOverviewCmd_ForceLiveInLiveAccountIsANoOp(t *testing.T) {
	opts := stubOverviewGatewayCapturingOptions(t, newIntegHlthGateway(t, http.StatusOK, integHlthAlertsBody, nil))
	stubOverviewDefaultAPIVersion(t, "")
	calls := stubOverviewSandboxParent(t, "", true)

	out, err := runOverview(t, "--force-live")
	require.NoError(t, err)

	assert.Equal(t, overviewDPMAction, out)
	assert.Empty(t, *calls)
	assert.Equal(t, "acct_live", opts.Credentials.OAKContext)
}

func TestOverviewCmd_LiveFromSandboxPointsToForceLive(t *testing.T) {
	stubOverviewDefaultAPIVersion(t, "")
	orig := resolveOverviewCredentials
	resolveOverviewCredentials = func(_ *config.Profile, live bool) (stripe.Credentials, error) {
		require.True(t, live)
		return stripe.Credentials{}, errorcategory.With(&config.ActiveContextLivemodeMismatchError{RequestedLivemode: true}, errorcategory.UserInput)
	}
	t.Cleanup(func() { resolveOverviewCredentials = orig })

	_, err := runOverview(t, "--live")

	require.EqualError(t, err, "You're in a sandbox. Run `stripe overview --force-live` to see insights for your live account.")
}

func TestSandboxParentAccount(t *testing.T) {
	serve := func(t *testing.T, status int, body string) string {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/stripecli/context_statuses", r.URL.Path)
			assert.NoError(t, r.ParseForm())
			assert.Equal(t, "acct_sandbox", r.Form.Get("contexts[0][account_id]"))
			assert.Equal(t, "false", r.Form.Get("contexts[0][livemode]"))
			w.WriteHeader(status)
			_, _ = w.Write([]byte(body))
		}))
		t.Cleanup(server.Close)
		return server.URL
	}

	t.Run("authorized parent", func(t *testing.T) {
		url := serve(t, http.StatusOK, `{"contexts": [{"account_id": "acct_sandbox", "livemode": false, "valid": true, "parent_account_id": "acct_live"}]}`)
		parent, ok := sandboxParentAccount(context.Background(), url, nil, sandboxCreds)
		assert.True(t, ok)
		assert.Equal(t, "acct_live", parent)
	})

	t.Run("parent not authorized", func(t *testing.T) {
		url := serve(t, http.StatusOK, `{"contexts": [{"account_id": "acct_sandbox", "livemode": false, "valid": true}]}`)
		parent, ok := sandboxParentAccount(context.Background(), url, nil, sandboxCreds)
		assert.True(t, ok)
		assert.Empty(t, parent)
	})

	t.Run("lookup fails", func(t *testing.T) {
		url := serve(t, http.StatusInternalServerError, `{"error": {"message": "boom"}}`)
		_, ok := sandboxParentAccount(context.Background(), url, nil, sandboxCreds)
		assert.False(t, ok)
	})
}
