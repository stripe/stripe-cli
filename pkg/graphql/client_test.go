package graphql

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

const testAPIKey = "sk_test_1234567890abcdef"

func newGatewayServer(t *testing.T, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, defaultPath, r.URL.Path)
		check(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"v2ListIntegHlthAlerts":{"alerts":[{"health_insight_id":"hi_123","status":"open"}],"next_page":"page_2"}}}`))
	}))
	t.Cleanup(ts.Close)
	return ts
}

func TestNewClientSendsCredentialsAndPreviewVersion(t *testing.T) {
	ts := newGatewayServer(t, func(r *http.Request) {
		assert.Equal(t, "Bearer "+testAPIKey, r.Header.Get("Authorization"))
		assert.Equal(t, requests.StripePreviewVersionHeaderValue, r.Header.Get("Stripe-Version"))
		assert.Empty(t, r.Header.Get("Stripe-Context"))

		var body struct {
			OperationName string                 `json:"operationName"`
			Variables     map[string]interface{} `json:"variables"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, "ListIntegHlthAlerts", body.OperationName)
		// Unset optional filters are omitted rather than sent as zero values.
		assert.Equal(t, map[string]interface{}{"statuses": []interface{}{"open"}}, body.Variables)
	})

	client, err := NewClient(context.Background(), Options{
		Profile:   &config.Profile{APIKey: testAPIKey},
		Host:      strings.TrimPrefix(ts.URL, "https://"),
		Transport: ts.Client().Transport,
	})
	require.NoError(t, err)

	resp, err := ListIntegHlthAlerts(context.Background(), client, []string{"open"}, 0, "", "", "")
	require.NoError(t, err)
	require.Len(t, resp.V2ListIntegHlthAlerts.Alerts, 1)
	assert.Equal(t, "hi_123", resp.V2ListIntegHlthAlerts.Alerts[0].Health_insight_id)
	assert.Equal(t, "open", resp.V2ListIntegHlthAlerts.Alerts[0].Status)
	assert.Equal(t, "page_2", resp.V2ListIntegHlthAlerts.Next_page)
}

func TestNewClientUsesStripeVersionOverride(t *testing.T) {
	ts := newGatewayServer(t, func(r *http.Request) {
		assert.Equal(t, "unsafe-development", r.Header.Get("Stripe-Version"))
	})

	client, err := NewClient(context.Background(), Options{
		Profile:       &config.Profile{APIKey: testAPIKey},
		Host:          strings.TrimPrefix(ts.URL, "https://"),
		StripeVersion: "unsafe-development",
		Transport:     ts.Client().Transport,
	})
	require.NoError(t, err)

	_, err = ListIntegHlthAlerts(context.Background(), client, []string{"open"}, 0, "", "", "")
	require.NoError(t, err)
}

func TestNewClientRequiresProfile(t *testing.T) {
	_, err := NewClient(context.Background(), Options{})
	require.Error(t, err)
}

func TestWithWorkspaceContext(t *testing.T) {
	stubWorkspace := func(t *testing.T, wc requests.WorkspaceContext, err error) {
		t.Helper()
		orig := getWorkspaceContext
		getWorkspaceContext = func(context.Context, string, *config.Profile, stripe.Credentials, bool) (requests.WorkspaceContext, error) {
			return wc, err
		}
		t.Cleanup(func() { getWorkspaceContext = orig })
	}
	opts := Options{Profile: &config.Profile{}, APIBaseURL: stripe.DefaultAPIBaseURL, Livemode: true}
	oak := stripe.NewOAKCredentials("oak_live_123", "acct_123", true)

	t.Run("replaces the account with its workspace", func(t *testing.T) {
		stubWorkspace(t, requests.WorkspaceContext{WorkspaceID: "wksp_456"}, nil)

		creds, err := withWorkspaceContext(context.Background(), opts, oak)
		require.NoError(t, err)
		assert.Equal(t, "wksp_456", creds.OAKContext)
		assert.Equal(t, "oak_live_123", creds.Token)
		require.NotNil(t, creds.OAKLivemode)
		assert.True(t, *creds.OAKLivemode)
	})

	t.Run("leaves API key credentials unchanged", func(t *testing.T) {
		stubWorkspace(t, requests.WorkspaceContext{}, errors.New("should not be called"))

		apiKey := stripe.NewAPIKeyCredentials("sk_test_123")
		creds, err := withWorkspaceContext(context.Background(), opts, apiKey)
		require.NoError(t, err)
		assert.Equal(t, apiKey, creds)
	})

	t.Run("returns lookup errors", func(t *testing.T) {
		stubWorkspace(t, requests.WorkspaceContext{}, errors.New("boom"))

		_, err := withWorkspaceContext(context.Background(), opts, oak)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "acct_123")
		assert.Contains(t, err.Error(), "boom")
	})

	t.Run("rejects an empty workspace", func(t *testing.T) {
		stubWorkspace(t, requests.WorkspaceContext{}, nil)

		_, err := withWorkspaceContext(context.Background(), opts, oak)
		require.Error(t, err)
	})
}

func TestHostForAccessBaseURL(t *testing.T) {
	assert.Equal(t, QAHost, HostForAccessBaseURL(login.QAAccessBaseURL))
	assert.Equal(t, DefaultHost, HostForAccessBaseURL(login.DefaultAccessBaseURL))
}
