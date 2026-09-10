package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

func TestManagementClientListAccessible(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		require.Equal(t, "acct_live_123", r.Header.Get("Stripe-Context"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		switch r.URL.Path {
		case "/v1/stripecli/workspace_context":
			require.Empty(t, r.URL.RawQuery)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Empty(t, body)
			_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
		case accessibleSandboxesPath:
			require.Equal(t, "check_user_sandbox_management_actions=false&include_is_dashboard_accessible=false&include_legacy_testmode=false&live_compartment_parent_id=wksp_live_parent&recursively_resolve=false", r.URL.RawQuery)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Empty(t, body)
			_, _ = w.Write([]byte(`{
  "workspaces": [
    {"id":"wksp_test_z","merchant_id":"acct_z","name":"Zeta","access_level":3},
    {"id":"wksp_test_b","merchant_id":"acct_b","name":"Alpha","access_level":1}
  ],
  "organizations": [{"workspaces": [
    {"id":"wksp_test_a","merchant_id":"acct_a","name":"alpha","access_level":2},
    {"id":"wksp_test_b","merchant_id":"acct_b","name":"Alpha","access_level":1}
  ]}]
}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewManagementClient(server.URL, profile)
	got, err := client.ListAccessible(context.Background())
	require.NoError(t, err)
	require.Equal(t, []ManagedSandbox{
		{WorkspaceID: "wksp_test_a", AccountID: "acct_a", Name: "alpha", AccessLevel: AccessLevelSandboxChildren},
		{WorkspaceID: "wksp_test_b", AccountID: "acct_b", Name: "Alpha", AccessLevel: AccessLevelDirect},
		{WorkspaceID: "wksp_test_z", AccountID: "acct_z", Name: "Zeta", AccessLevel: AccessLevelNone},
	}, got)
}

func TestManagementClientListAccessibleEmpty(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
	server := managementTestServer(t, `{"workspace_id":"wksp_live_parent"}`, `{"workspaces":[],"organizations":[]}`)
	defer server.Close()

	got, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Empty(t, got)
}

func TestManagementClientReusesProactivelyRefreshedCredentials(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_initial")
	require.NoError(t, config.SaveUATExpiresAt(time.Now().Add(30*time.Second)))
	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		p.UAT = "oak_refreshed"
		return nil
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "Bearer oak_refreshed", r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/v1/stripecli/workspace_context":
			_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
		case accessibleSandboxesPath:
			_, _ = w.Write([]byte(`{"workspaces":[],"organizations":[]}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	_, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, refreshes)
}

func TestManagementClientRequiresActiveLiveOAuth(t *testing.T) {
	tests := []struct {
		name        string
		profile     func(*testing.T) *config.Profile
		category    errorcategory.Category
		messagePart string
	}{
		{
			name: "api key fallback",
			profile: func(t *testing.T) *config.Profile {
				profile := managementTestProfile(t, "acct_live_123", true, "")
				profile.APIKey = "sk_live_12345"
				return profile
			},
			category: errorcategory.Auth,
		},
		{
			name: "testmode context",
			profile: func(t *testing.T) *config.Profile {
				return managementTestProfile(t, "acct_test_123", false, "oak_test_123")
			},
			category:    errorcategory.UserInput,
			messagePart: "switch context",
		},
		{
			name: "missing account context",
			profile: func(t *testing.T) *config.Profile {
				return managementTestProfile(t, "", true, "oak_test_123")
			},
			category: errorcategory.Auth,
		},
		{
			name: "organization context",
			profile: func(t *testing.T) *config.Profile {
				return managementTestProfile(t, "org_123", true, "oak_test_123")
			},
			category: errorcategory.Auth,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := test.profile(t)
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			defer server.Close()

			got, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
			require.Error(t, err)
			require.Nil(t, got)
			assert.Equal(t, test.category, mustErrorCategory(t, err))
			if test.messagePart != "" {
				assert.Contains(t, err.Error(), test.messagePart)
			}
			assert.Equal(t, 0, requests)
			assert.NotContains(t, err.Error(), "oak_")
			assert.NotContains(t, err.Error(), "acct_")
		})
	}
}

func TestManagementClientRejectsInvalidWorkspaceContext(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "malformed JSON", response: `{"workspace_id":`},
		{name: "missing workspace", response: `{}`},
		{name: "not a workspace", response: `{"workspace_id":"acct_live_123"}`},
		{name: "testmode workspace", response: `{"workspace_id":"wksp_test_123"}`},
		{name: "prefix without an identifier", response: `{"workspace_id":"wksp_"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				require.Equal(t, "/v1/stripecli/workspace_context", r.URL.Path)
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()

			got, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
			require.Error(t, err)
			require.Nil(t, got)
			assert.Equal(t, errorcategory.API, mustErrorCategory(t, err))
			assert.Equal(t, 1, requests)
			assert.NotContains(t, err.Error(), "wksp_")
			assert.NotContains(t, err.Error(), "acct_")
		})
	}
}

func TestManagementClientRejectsInvalidSandboxResponses(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "malformed JSON", response: `{"workspaces":`},
		{name: "missing workspace id", response: `{"workspaces":[{"merchant_id":"acct_1","name":"one","access_level":1}]}`},
		{name: "missing account id", response: `{"workspaces":[{"id":"wksp_test_1","name":"one","access_level":1}]}`},
		{name: "missing name", response: `{"workspaces":[{"id":"wksp_test_1","merchant_id":"acct_1","access_level":1}]}`},
		{name: "live workspace", response: `{"workspaces":[{"id":"wksp_live_1","merchant_id":"acct_1","name":"one","access_level":1}]}`},
		{name: "zero access level", response: `{"workspaces":[{"id":"wksp_test_1","merchant_id":"acct_1","name":"one","access_level":0}]}`},
		{name: "unknown access level", response: `{"workspaces":[{"id":"wksp_test_1","merchant_id":"acct_1","name":"one","access_level":4}]}`},
		{name: "string access level", response: `{"workspaces":[{"id":"wksp_test_1","merchant_id":"acct_1","name":"one","access_level":"DIRECT"}]}`},
		{name: "conflicting duplicate", response: `{"workspaces":[
  {"id":"wksp_test_1","merchant_id":"acct_1","name":"one","access_level":1},
  {"id":"wksp_test_1","merchant_id":"acct_2","name":"one","access_level":1}
]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
			server := managementTestServer(t, `{"workspace_id":"wksp_live_parent"}`, test.response)
			defer server.Close()

			got, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
			require.Error(t, err)
			require.Nil(t, got)
			assert.Equal(t, errorcategory.API, mustErrorCategory(t, err))
			assert.NotContains(t, err.Error(), "sentinel")
			assert.NotContains(t, err.Error(), "wksp_")
			assert.NotContains(t, err.Error(), "acct_")
		})
	}
}

func TestManagementClientSanitizesDependencyErrors(t *testing.T) {
	tests := []struct {
		name     string
		stage    string
		status   int
		category errorcategory.Category
	}{
		{name: "M2 unauthorized", stage: "M2", status: http.StatusUnauthorized, category: errorcategory.Auth},
		{name: "M2 forbidden", stage: "M2", status: http.StatusForbidden, category: errorcategory.Auth},
		{name: "M2 not found", stage: "M2", status: http.StatusNotFound, category: errorcategory.API},
		{name: "M2 rate limited", stage: "M2", status: http.StatusTooManyRequests, category: errorcategory.RateLimit},
		{name: "M2 backend failure", stage: "M2", status: http.StatusBadGateway, category: errorcategory.API},
		{name: "GUAS unauthorized", stage: "GUAS", status: http.StatusUnauthorized, category: errorcategory.Auth},
		{name: "GUAS forbidden", stage: "GUAS", status: http.StatusForbidden, category: errorcategory.Auth},
		{name: "GUAS not found", stage: "GUAS", status: http.StatusNotFound, category: errorcategory.API},
		{name: "GUAS rate limited", stage: "GUAS", status: http.StatusTooManyRequests, category: errorcategory.RateLimit},
		{name: "GUAS backend failure", stage: "GUAS", status: http.StatusBadGateway, category: errorcategory.API},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if test.stage == "M2" || r.URL.Path == "/v1/stripecli/workspace_context" {
					if test.stage == "M2" {
						w.WriteHeader(test.status)
						_, _ = w.Write([]byte(`{"error":{"message":"sentinel response acct_secret wksp_secret"}}`))
						return
					}
					_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
					return
				}

				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(`{"error":{"message":"sentinel response acct_secret wksp_secret"}}`))
			}))
			defer server.Close()

			got, err := NewManagementClient(server.URL, profile).ListAccessible(context.Background())
			require.Error(t, err)
			require.Nil(t, got)
			assert.Equal(t, test.category, mustErrorCategory(t, err))
			assert.NotContains(t, err.Error(), "sentinel")
			assert.NotContains(t, err.Error(), "acct_secret")
			assert.NotContains(t, err.Error(), "wksp_secret")
		})
	}
}

func TestManagementClientSanitizesTransportErrors(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
	server := httptest.NewServer(http.NotFoundHandler())
	serverURL := server.URL
	server.Close()

	got, err := NewManagementClient(serverURL, profile).ListAccessible(context.Background())
	require.Error(t, err)
	require.Nil(t, got)
	require.Equal(t, errorcategory.Network, mustErrorCategory(t, err))
	assert.NotContains(t, err.Error(), serverURL)
}

func TestManagementClientPrepareDelete(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
	server := managementTestServer(t, `{"workspace_id":"wksp_live_parent"}`, `{
  "workspaces": [
    {"id":"wksp_test_other","merchant_id":"acct_other","name":"Other","access_level":1},
    {"id":"wksp_test_target","merchant_id":"acct_target","name":"Target","access_level":2}
  ]
}`)
	defer server.Close()

	target, err := NewManagementClient(server.URL, profile).PrepareDelete(context.Background(), "acct_target")
	require.NoError(t, err)
	require.NotNil(t, target)
	assert.Equal(t, "acct_target", target.AccountID)
	assert.Equal(t, "Target", target.Name)
	assert.Equal(t, "wksp_test_target", target.workspaceID)
	assert.Equal(t, "acct_live_parent", target.liveAccountID)
}

func TestManagementClientPrepareDeleteRejectsInvalidOrMissingAccount(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	for _, accountID := range []string{"", "acct_", "org_target", "wksp_test_target"} {
		target, err := NewManagementClient(server.URL, profile).PrepareDelete(context.Background(), accountID)
		require.Error(t, err)
		assert.Nil(t, target)
		assert.Equal(t, errorcategory.UserInput, mustErrorCategory(t, err))
	}

	assert.Zero(t, requests)
}

func TestManagementClientPrepareDeleteRejectsMissingAndAmbiguousMatches(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "missing",
			response: `{"workspaces":[{"id":"wksp_test_other","merchant_id":"acct_other","name":"Other","access_level":1}]}`,
		},
		{
			name: "ambiguous",
			response: `{"workspaces":[
  {"id":"wksp_test_one","merchant_id":"acct_target","name":"One","access_level":1},
  {"id":"wksp_test_two","merchant_id":"acct_target","name":"Two","access_level":1}
]}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
			server := managementTestServer(t, `{"workspace_id":"wksp_live_parent"}`, test.response)
			defer server.Close()

			target, err := NewManagementClient(server.URL, profile).PrepareDelete(context.Background(), "acct_target")
			require.Error(t, err)
			assert.Nil(t, target)
			assert.Equal(t, errorcategory.UserInput, mustErrorCategory(t, err))
			assert.NotContains(t, err.Error(), "acct_target")
			assert.NotContains(t, err.Error(), "wksp_test")
		})
	}
}

func TestManagementClientDeleteUsesDirectSandboxTestmodeCredentials(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
	target := &DeleteTarget{
		AccountID:     "acct_target",
		Name:          "Target",
		workspaceID:   "wksp_test_target",
		liveAccountID: "acct_live_parent",
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/workspaces/undocumented/testmode/wksp_test_target/close", r.URL.Path)
		require.Equal(t, "Bearer oak_live_token", r.Header.Get("Authorization"))
		require.Equal(t, "acct_target", r.Header.Get("Stripe-Context"))
		require.Equal(t, "false", r.Header.Get("Stripe-Livemode"))
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Empty(t, body)
		_, _ = w.Write([]byte(`{"id":"wksp_test_target"}`))
	}))
	defer server.Close()

	err := NewManagementClient(server.URL, profile).Delete(context.Background(), target)
	require.NoError(t, err)
}

func TestManagementClientDeleteRejectsChangedLiveContextBeforeClose(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
	target := &DeleteTarget{
		AccountID:     "acct_target",
		Name:          "Target",
		workspaceID:   "wksp_test_target",
		liveAccountID: "acct_live_parent",
	}
	require.NoError(t, config.SaveActiveContext("acct_other_live_parent", true))
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	err := NewManagementClient(server.URL, profile).Delete(context.Background(), target)
	require.Error(t, err)
	assert.Equal(t, errorcategory.UserInput, mustErrorCategory(t, err))
	assert.Contains(t, err.Error(), "active live account changed")
	assert.NotContains(t, err.Error(), "acct_")
	assert.Zero(t, requests)
}

func TestManagementClientDeleteRefreshesAndRetriesOnce(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	target := &DeleteTarget{
		AccountID:     "acct_target",
		Name:          "Target",
		workspaceID:   "wksp_test_target",
		liveAccountID: "acct_live_parent",
	}
	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_refreshed"), "test refreshed token")
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "acct_target", r.Header.Get("Stripe-Context"))
		require.Equal(t, "false", r.Header.Get("Stripe-Livemode"))
		if requests == 1 {
			require.Equal(t, "Bearer oak_initial", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"sentinel oak_initial wksp_test_target"}}`))
			return
		}
		require.Equal(t, "Bearer oak_refreshed", r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"id":"wksp_test_target"}`))
	}))
	defer server.Close()

	err := NewManagementClient(server.URL, profile).Delete(context.Background(), target)
	require.NoError(t, err)
	assert.Equal(t, 1, refreshes)
	assert.Equal(t, 2, requests)
}

func TestManagementClientDeleteDoesNotRetryTwice(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	target := &DeleteTarget{
		AccountID:     "acct_target",
		Name:          "Target",
		workspaceID:   "wksp_test_target",
		liveAccountID: "acct_live_parent",
	}
	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte("oak_refreshed"), "test refreshed token")
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"sentinel oak_secret wksp_test_target"}}`))
	}))
	defer server.Close()

	err := NewManagementClient(server.URL, profile).Delete(context.Background(), target)
	require.Error(t, err)
	assert.Equal(t, 1, refreshes)
	assert.Equal(t, 2, requests)
	assert.NotContains(t, err.Error(), "sentinel")
	assert.NotContains(t, err.Error(), "oak_")
	assert.NotContains(t, err.Error(), "wksp_test")
}

func TestManagementClientDeleteSanitizesFailures(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "forbidden", statusCode: http.StatusForbidden, body: `{"error":{"message":"sentinel acct_target wksp_test_target"}}`},
		{name: "malformed success", statusCode: http.StatusOK, body: `{"id":`},
		{name: "mismatched success", statusCode: http.StatusOK, body: `{"id":"wksp_test_other"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_parent", true, "oak_live_token")
			target := &DeleteTarget{
				AccountID:     "acct_target",
				Name:          "Target",
				workspaceID:   "wksp_test_target",
				liveAccountID: "acct_live_parent",
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()

			err := NewManagementClient(server.URL, profile).Delete(context.Background(), target)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "sentinel")
			assert.NotContains(t, err.Error(), "acct_target")
			assert.NotContains(t, err.Error(), "wksp_test")
			assert.NotContains(t, err.Error(), server.URL)
		})
	}
}

func managementTestProfile(t *testing.T, accountID string, livemode bool, token string) *config.Profile {
	t.Helper()
	activeContext, err := json.Marshal(config.ActiveContext{AccountID: accountID, Livemode: livemode})
	require.NoError(t, err)

	previousKeyRing := config.KeyRing
	previousRefresher := config.OAuthTokenRefresher
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte(token),
		config.OAuthActiveContextKeychainKey: activeContext,
	})
	config.OAuthTokenRefresher = nil
	t.Cleanup(func() {
		config.KeyRing = previousKeyRing
		config.OAuthTokenRefresher = previousRefresher
	})

	return &config.Profile{ProfileName: "default"}
}

func managementTestServer(t *testing.T, workspaceResponse, sandboxesResponse string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/stripecli/workspace_context":
			_, _ = w.Write([]byte(workspaceResponse))
		case accessibleSandboxesPath:
			_, _ = w.Write([]byte(sandboxesResponse))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
}

func mustErrorCategory(t *testing.T, err error) errorcategory.Category {
	t.Helper()
	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	return category
}
