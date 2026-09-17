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

func TestManagementClientCreateCopyLive(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
	requestsSeen := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestsSeen = append(requestsSeen, r.Method+" "+r.URL.Path)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		switch r.URL.Path {
		case "/v1/stripecli/playground_context":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "acct_live_123", r.Header.Get("Stripe-Context"))
			_, _ = w.Write([]byte(`{"playground_id":"play_parent"}`))
		case "/v1/stripecli/workspace_context":
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "acct_live_123", r.Header.Get("Stripe-Context"))
			_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
		case "/v2/sandboxes":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "play_parent", r.Header.Get("Stripe-Context"))
			require.Empty(t, r.Header.Get("Stripe-Account"))
			require.Equal(t, "application/json", r.Header.Get("Content-Type"))
			require.NotEmpty(t, r.Header.Get("Idempotency-Key"))

			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]interface{}{
				"name":             "Copied sandbox",
				"activate_sandbox": true,
				"replica_of":       "wksp_live_parent",
			}, body)
			for _, excluded := range []string{"target_compartment", "idempotency_token", "access_level", "objects"} {
				require.NotContains(t, body, excluded)
			}
			_, _ = w.Write([]byte(`{"id":"wksp_test_internal","v1_account_id":"acct_created"}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	created, err := NewManagementClient(server.URL, profile).Create(context.Background(), CreateOptions{Name: "Copied sandbox"})
	require.NoError(t, err)
	require.Equal(t, CreatedSandbox{AccountID: "acct_created"}, created)
	require.Equal(t, []string{
		"GET /v1/stripecli/playground_context",
		"GET /v1/stripecli/workspace_context",
		"POST /v2/sandboxes",
	}, requestsSeen)
}

func TestManagementClientCreateBlank(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/stripecli/playground_context":
			_, _ = w.Write([]byte(`{"playground_id":"play_parent"}`))
		case "/v2/sandboxes":
			var body map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]interface{}{
				"name":              "Blank sandbox",
				"activate_sandbox":  false,
				"business_location": "CA",
			}, body)
			require.NotContains(t, body, "replica_of")
			_, _ = w.Write([]byte(`{"v1_account_id":"acct_blank"}`))
		case "/v1/stripecli/workspace_context":
			t.Fatal("blank creation must not resolve the live workspace")
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	created, err := NewManagementClient(server.URL, profile).Create(context.Background(), CreateOptions{
		Name:    "Blank sandbox",
		Blank:   true,
		Country: "CA",
	})
	require.NoError(t, err)
	require.Equal(t, CreatedSandbox{AccountID: "acct_blank"}, created)
}

func TestManagementClientCreateRejectsInvalidIdentifiers(t *testing.T) {
	tests := []struct {
		name               string
		playgroundResponse string
		workspaceResponse  string
		createResponse     string
		wantRequests       int
	}{
		{name: "playground", playgroundResponse: `{"playground_id":"acct_secret"}`, wantRequests: 1},
		{name: "workspace", playgroundResponse: `{"playground_id":"play_parent"}`, workspaceResponse: `{"workspace_id":"wksp_test_secret"}`, wantRequests: 2},
		{name: "created account", playgroundResponse: `{"playground_id":"play_parent"}`, workspaceResponse: `{"workspace_id":"wksp_live_parent"}`, createResponse: `{"v1_account_id":"play_secret"}`, wantRequests: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_123", true, "oak_test_123")
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				switch r.URL.Path {
				case "/v1/stripecli/playground_context":
					_, _ = w.Write([]byte(test.playgroundResponse))
				case "/v1/stripecli/workspace_context":
					_, _ = w.Write([]byte(test.workspaceResponse))
				case "/v2/sandboxes":
					_, _ = w.Write([]byte(test.createResponse))
				default:
					t.Fatalf("unexpected request path %q", r.URL.Path)
				}
			}))
			defer server.Close()

			created, err := NewManagementClient(server.URL, profile).Create(context.Background(), CreateOptions{Name: "test"})
			require.Error(t, err)
			require.Empty(t, created)
			require.Equal(t, errorcategory.API, mustErrorCategory(t, err))
			require.Equal(t, test.wantRequests, requestCount)
			require.NotContains(t, err.Error(), "secret")
			require.NotContains(t, err.Error(), "play_")
			require.NotContains(t, err.Error(), "wksp_")
		})
	}
}

func TestManagementClientCreateRetryPreservesTargetAndIdempotency(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_123", true, "oak_initial")
	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		p.UAT = "oak_refreshed"
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte(p.UAT), "refreshed test token")
	}

	postHeaders := make([]http.Header, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/stripecli/playground_context":
			_, _ = w.Write([]byte(`{"playground_id":"play_parent"}`))
		case "/v1/stripecli/workspace_context":
			_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
		case "/v2/sandboxes":
			postHeaders = append(postHeaders, r.Header.Clone())
			if len(postHeaders) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"expired"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"v1_account_id":"acct_created"}`))
		default:
			t.Fatalf("unexpected request path %q", r.URL.Path)
		}
	}))
	defer server.Close()

	created, err := NewManagementClient(server.URL, profile).Create(context.Background(), CreateOptions{Name: "Retry sandbox"})
	require.NoError(t, err)
	require.Equal(t, CreatedSandbox{AccountID: "acct_created"}, created)
	require.Equal(t, 1, refreshes)
	require.Len(t, postHeaders, 2)
	require.Equal(t, "Bearer oak_initial", postHeaders[0].Get("Authorization"))
	require.Equal(t, "Bearer oak_refreshed", postHeaders[1].Get("Authorization"))
	require.Equal(t, "play_parent", postHeaders[0].Get("Stripe-Context"))
	require.Equal(t, "play_parent", postHeaders[1].Get("Stripe-Context"))
	require.NotEmpty(t, postHeaders[0].Get("Idempotency-Key"))
	require.Equal(t, postHeaders[0].Get("Idempotency-Key"), postHeaders[1].Get("Idempotency-Key"))
}

func TestManagementClientDelete(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	var closeHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[
  {"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"},
  {"id":"wksp_test_other","merchant_id":"acct_other","name":"Other sandbox"}
]}`) {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/workspaces/undocumented/testmode/wksp_test_target/close", r.URL.EscapedPath())
		closeHeaders = r.Header.Clone()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Empty(t, body)
		_, _ = w.Write([]byte(`{"id":"wksp_test_target"}`))
	}))
	defer server.Close()

	deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "  acct_target  ")
	require.NoError(t, err)
	require.Equal(t, DeletedSandbox{AccountID: "acct_target", Name: "Target sandbox"}, deleted)
	require.Equal(t, "Bearer oak_initial", closeHeaders.Get("Authorization"))
	require.Equal(t, "acct_target", closeHeaders.Get("Stripe-Context"))
	require.Equal(t, "false", closeHeaders.Get("Stripe-Livemode"))
	require.Empty(t, closeHeaders.Get("Stripe-Account"))

	activeContext, err := config.GetActiveContext()
	require.NoError(t, err)
	require.Equal(t, &config.ActiveContext{AccountID: "acct_live_parent", Livemode: true}, activeContext)
}

func TestManagementClientDeleteRefreshPreservesTarget(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		p.UAT = "oak_refreshed"
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte(p.UAT), "refreshed test token")
	}

	closeHeaders := make([]http.Header, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"}]}`) {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		closeHeaders = append(closeHeaders, r.Header.Clone())
		if len(closeHeaders) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"expired"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"wksp_test_target"}`))
	}))
	defer server.Close()

	deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "acct_target")
	require.NoError(t, err)
	require.Equal(t, DeletedSandbox{AccountID: "acct_target", Name: "Target sandbox"}, deleted)
	require.Equal(t, 1, refreshes)
	require.Len(t, closeHeaders, 2)
	for i, headers := range closeHeaders {
		wantToken := "oak_initial"
		if i == 1 {
			wantToken = "oak_refreshed"
		}
		require.Equal(t, "Bearer "+wantToken, headers.Get("Authorization"))
		require.Equal(t, "acct_target", headers.Get("Stripe-Context"))
		require.Equal(t, "false", headers.Get("Stripe-Livemode"))
		require.Empty(t, headers.Get("Stripe-Account"))
	}

	activeContext, err := config.GetActiveContext()
	require.NoError(t, err)
	require.Equal(t, &config.ActiveContext{AccountID: "acct_live_parent", Livemode: true}, activeContext)
}

func TestManagementClientDeleteRejectsInvalidAccounts(t *testing.T) {
	for _, accountID := range []string{"", "org_123", "acct_", "not_an_account"} {
		t.Run(accountID, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			defer server.Close()

			deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), accountID)
			require.Error(t, err)
			require.Empty(t, deleted)
			require.Equal(t, errorcategory.UserInput, mustErrorCategory(t, err))
			require.Equal(t, 0, requests)
		})
	}
}

func TestManagementClientDeleteDoesNotSearchOutsideActiveParent(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	closeRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[{"id":"wksp_test_other_parent","merchant_id":"acct_other","name":"Other sandbox"}]}`) {
			return
		}
		closeRequested = true
	}))
	defer server.Close()

	deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "acct_target")
	require.Error(t, err)
	require.Empty(t, deleted)
	require.Equal(t, errorcategory.API, mustErrorCategory(t, err))
	require.Contains(t, err.Error(), "sandbox list")
	require.False(t, closeRequested)
}

func TestManagementClientDeleteRejectsAmbiguousTarget(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	closeRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[
  {"id":"wksp_test_a","merchant_id":"acct_target","name":"First"},
  {"id":"wksp_test_b","merchant_id":"acct_target","name":"Second"}
]}`) {
			return
		}
		closeRequested = true
	}))
	defer server.Close()

	deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "acct_target")
	require.Error(t, err)
	require.Empty(t, deleted)
	require.Equal(t, errorcategory.API, mustErrorCategory(t, err))
	require.Contains(t, err.Error(), "sandbox list")
	require.False(t, closeRequested)
}

func TestManagementClientDeleteSanitizesHTTPFailures(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		category errorcategory.Category
	}{
		{name: "unauthorized", status: http.StatusUnauthorized, category: errorcategory.Auth},
		{name: "forbidden", status: http.StatusForbidden, category: errorcategory.Auth},
		{name: "rate limited", status: http.StatusTooManyRequests, category: errorcategory.RateLimit},
		{name: "server error", status: http.StatusBadGateway, category: errorcategory.API},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"}]}`) {
					return
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(`{"error":{"message":"sentinel oak_secret acct_secret wksp_secret"}}`))
			}))
			defer server.Close()

			deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "acct_target")
			require.Error(t, err)
			require.Empty(t, deleted)
			require.Equal(t, test.category, mustErrorCategory(t, err))
			require.NotContains(t, err.Error(), "sentinel")
			require.NotContains(t, err.Error(), "oak_")
			require.NotContains(t, err.Error(), "acct_")
			require.NotContains(t, err.Error(), "wksp_")
		})
	}
}

func TestManagementClientDeleteRequiresConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{name: "malformed response", response: `{"id":`},
		{name: "mismatched workspace", response: `{"id":"wksp_test_other"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"}]}`) {
					return
				}
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()

			deleted, err := NewManagementClient(server.URL, profile).Delete(context.Background(), "acct_target")
			require.Error(t, err)
			require.Empty(t, deleted)
			require.Equal(t, errorcategory.API, mustErrorCategory(t, err))
			require.Contains(t, err.Error(), "sandbox list")
			require.NotContains(t, err.Error(), "wksp_")
		})
	}
}

func TestManagementClientDeleteTransportFailureIsAmbiguous(t *testing.T) {
	profile := managementTestProfile(t, "acct_live_parent", true, "oak_initial")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveDeleteDiscoveryResponse(t, w, r, "acct_live_parent", "oak_initial", `{"workspaces":[{"id":"wksp_test_target","merchant_id":"acct_target","name":"Target sandbox"}]}`) {
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	deleted, err := NewManagementClient(server.URL, profile).Delete(ctx, "acct_target")
	require.Error(t, err)
	require.Empty(t, deleted)
	require.Equal(t, errorcategory.Network, mustErrorCategory(t, err))
	require.Contains(t, err.Error(), "sandbox list")
}

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
    {"id":"wksp_test_z","merchant_id":"acct_z","name":"Zeta","access_level":"direct_access","compartment_labels":[]},
    {"id":"wksp_test_b","merchant_id":"acct_b","name":"Alpha","access_level":"direct_access","compartment_labels":[{"usage_type":"sandbox_access_level_global"}]}
  ],
  "organizations": [{
    "access_level":"direct_access",
    "compartment_labels":[{"usage_type":"sandbox_access_level_global"}],
    "workspaces": [
    {"id":"wksp_test_a","merchant_id":"acct_a","name":"alpha","access_level":"direct_access","compartment_labels":[{"usage_type":"sandbox_access_level_private"}]},
    {"id":"wksp_test_b","merchant_id":"acct_b","name":"Alpha","access_level":"direct_access","compartment_labels":[{"usage_type":"sandbox_access_level_private"}]}
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
		{WorkspaceID: "wksp_test_a", AccountID: "acct_a", Name: "alpha", AccessLevel: SandboxAccessLevelGlobal},
		{WorkspaceID: "wksp_test_b", AccountID: "acct_b", Name: "Alpha", AccessLevel: SandboxAccessLevelGlobal},
		{WorkspaceID: "wksp_test_z", AccountID: "acct_z", Name: "Zeta", AccessLevel: SandboxAccessLevelPrivate},
	}, got)
}

func serveDeleteDiscoveryResponse(t *testing.T, w http.ResponseWriter, r *http.Request, liveAccount, token, sandboxesResponse string) bool {
	t.Helper()
	if r.Method != http.MethodGet {
		return false
	}
	require.Equal(t, http.MethodGet, r.Method)
	require.Equal(t, "Bearer "+token, r.Header.Get("Authorization"))
	require.Equal(t, liveAccount, r.Header.Get("Stripe-Context"))
	require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

	switch r.URL.Path {
	case "/v1/stripecli/workspace_context":
		_, _ = w.Write([]byte(`{"workspace_id":"wksp_live_parent"}`))
		return true
	case accessibleSandboxesPath:
		_, _ = w.Write([]byte(sandboxesResponse))
		return true
	default:
		return false
	}
}

func TestSandboxAccessLevelFromLabels(t *testing.T) {
	tests := []struct {
		name   string
		labels []compartmentLabel
		want   SandboxAccessLevel
	}{
		{name: "missing defaults to private", want: SandboxAccessLevelPrivate},
		{name: "explicit private", labels: []compartmentLabel{{UsageType: sandboxAccessLabelPrivate}}, want: SandboxAccessLevelPrivate},
		{name: "unrelated defaults to private", labels: []compartmentLabel{{UsageType: "testmode_app_developer"}}, want: SandboxAccessLevelPrivate},
		{name: "developer", labels: []compartmentLabel{{UsageType: sandboxAccessLabelDeveloper}}, want: SandboxAccessLevelDeveloper},
		{name: "global", labels: []compartmentLabel{{UsageType: sandboxAccessLabelGlobal}}, want: SandboxAccessLevelGlobal},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.want, sandboxAccessLevelFromLabels(test.labels))
		})
	}
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
		{name: "missing workspace id", response: `{"workspaces":[{"merchant_id":"acct_1","name":"one"}]}`},
		{name: "missing account id", response: `{"workspaces":[{"id":"wksp_test_1","name":"one"}]}`},
		{name: "missing name", response: `{"workspaces":[{"id":"wksp_test_1","merchant_id":"acct_1"}]}`},
		{name: "live workspace", response: `{"workspaces":[{"id":"wksp_live_1","merchant_id":"acct_1","name":"one"}]}`},
		{name: "conflicting duplicate", response: `{"workspaces":[
  {"id":"wksp_test_1","merchant_id":"acct_1","name":"one"},
  {"id":"wksp_test_1","merchant_id":"acct_2","name":"one"}
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
