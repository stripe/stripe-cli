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
)

func TestGetContextStatuses(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/stripecli/context_statuses", r.URL.Path)
		require.Equal(t, "Bearer oak_test_123", r.Header.Get("Authorization"))

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

	statuses, err := GetContextStatuses(context.Background(), ts.URL, nil, "oak_test_123", []ContextStatusQuery{
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
	statuses, err := GetContextStatuses(context.Background(), "http://unused.example", nil, "oak_test_123", nil)
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

	statuses, err := GetContextStatuses(context.Background(), ts.URL, nil, "oak_test_123", queries)
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

	_, err := GetContextStatuses(context.Background(), ts.URL, nil, "oak_test_123", []ContextStatusQuery{
		{AccountID: "acct_111", Livemode: true},
	})
	require.Error(t, err)
}
