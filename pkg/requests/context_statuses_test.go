package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func TestGetContextStatuses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/stripecli/context_statuses", r.URL.Path)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))
		// The API 403s without an OAK context identifying the caller's
		// active account/livemode, so these must always be sent.
		require.Equal(t, "acct_live", r.Header.Get("Stripe-Context"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"))

		require.NoError(t, r.ParseForm())
		require.Equal(t, "acct_111", r.Form.Get("contexts[0][account_id]"))
		require.Equal(t, "true", r.Form.Get("contexts[0][livemode]"))
		require.Equal(t, "acct_222", r.Form.Get("contexts[1][account_id]"))
		require.Equal(t, "false", r.Form.Get("contexts[1][livemode]"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"contexts": [
			{"account_id": "acct_111", "livemode": true, "valid": true},
			{"account_id": "acct_222", "livemode": false, "valid": true, "parent_account_id": "acct_111"}
		]}`)) //nolint:errcheck
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_live", true)
	statuses, err := GetContextStatuses(context.Background(), ts.URL, nil, creds, []ContextStatusQuery{
		{AccountID: "acct_111", Livemode: true},
		{AccountID: "acct_222", Livemode: false},
	})
	require.NoError(t, err)
	require.Equal(t, []ContextStatus{
		{AccountID: "acct_111", Livemode: true, Valid: true},
		{AccountID: "acct_222", Livemode: false, Valid: true, ParentAccountID: "acct_111"},
	}, statuses)
}

func TestGetContextStatuses_Empty(t *testing.T) {
	creds := stripe.NewOAKCredentials("oak_test_123", "acct_live", true)
	statuses, err := GetContextStatuses(context.Background(), "http://unused.example", nil, creds, nil)
	require.NoError(t, err)
	require.Nil(t, statuses)
}

func TestGetContextStatuses_Batches(t *testing.T) {
	var requestSizes []int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseForm())

		var accountIDs []string
		for key, vals := range r.Form {
			if strings.HasSuffix(key, "[account_id]") {
				accountIDs = append(accountIDs, vals[0])
			}
		}
		requestSizes = append(requestSizes, len(accountIDs))

		var contexts []string
		for _, id := range accountIDs {
			contexts = append(contexts, fmt.Sprintf(`{"account_id": %q, "livemode": false, "valid": true}`, id))
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"contexts": [%s]}`, strings.Join(contexts, ","))
	}))
	defer ts.Close()

	queries := make([]ContextStatusQuery, 150)
	for i := range queries {
		queries[i] = ContextStatusQuery{AccountID: "acct_" + strconv.Itoa(i), Livemode: false}
	}

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_live", true)
	statuses, err := GetContextStatuses(context.Background(), ts.URL, nil, creds, queries)
	require.NoError(t, err)
	require.Len(t, statuses, 150)
	require.ElementsMatch(t, []int{100, 50}, requestSizes)
}

// TestGetContextStatuses_RetriesWithLiveCredentialsAfterTokenRefresh exercises
// Base.performRequest's 401-refresh-and-retry path for a live active
// context. The retry re-resolves credentials via
// Profile.ResolveCredentials(rb.Livemode); if Livemode isn't set to match
// creds (as GetUserInfo already does), the retry fails with a mode-mismatch
// error instead of succeeding, silently disabling grouping.
func TestGetContextStatuses_RetriesWithLiveCredentialsAfterTokenRefresh(t *testing.T) {
	activeContext, err := json.Marshal(config.ActiveContext{AccountID: "acct_live", Livemode: true})
	require.NoError(t, err)

	origKeyRing, origRefresher := config.KeyRing, config.OAuthTokenRefresher
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oak_initial"),
		config.OAuthActiveContextKeychainKey: activeContext,
	})
	t.Cleanup(func() {
		config.KeyRing = origKeyRing
		config.OAuthTokenRefresher = origRefresher
	})

	refreshes := 0
	config.OAuthTokenRefresher = func(p *config.Profile) error {
		refreshes++
		p.UAT = "oak_refreshed"
		return config.KeyRing.Set(config.UATKeychainItemKey, []byte(p.UAT), "refreshed test token")
	}

	requests := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": {"code": "unauthorized", "message": "expired"}}`)) //nolint:errcheck
			return
		}

		require.Equal(t, "Bearer oak_refreshed", r.Header.Get("Authorization"))
		require.Equal(t, "true", r.Header.Get("Stripe-Livemode"), "the retry must resolve live credentials, matching the active context")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"contexts": [{"account_id": "acct_sandbox1", "livemode": false, "valid": true}]}`)) //nolint:errcheck
	}))
	defer ts.Close()

	profile := &config.Profile{ProfileName: "default"}
	creds := stripe.NewOAKCredentials("oak_initial", "acct_live", true)
	statuses, err := getContextStatusesBatch(context.Background(), ts.URL, profile, creds, []ContextStatusQuery{
		{AccountID: "acct_sandbox1", Livemode: false},
	})
	require.NoError(t, err)
	require.Equal(t, 1, refreshes)
	require.Equal(t, 2, requests)
	require.Equal(t, []ContextStatus{{AccountID: "acct_sandbox1", Livemode: false, Valid: true}}, statuses)
}

func TestGetContextStatuses_ErrOnStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "invalid token"}}`)) //nolint:errcheck
	}))
	defer ts.Close()

	creds := stripe.NewOAKCredentials("oak_test_123", "acct_live", true)
	_, err := GetContextStatuses(context.Background(), ts.URL, nil, creds, []ContextStatusQuery{
		{AccountID: "acct_111", Livemode: true},
	})
	require.Error(t, err)
}
