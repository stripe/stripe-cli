package requests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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
