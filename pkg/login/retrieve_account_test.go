package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const testName = "test_name"

func TestGetAccount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}
		account.Settings.Dashboard.DisplayName = testName

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	acc, err := GetUserAccount(context.Background(), ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		testName,
		acc.Settings.Dashboard.DisplayName,
	)
	require.Equal(
		t,
		"acct_123",
		acc.ID,
	)
}

func TestGetAccountNoDisplayName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	acc, err := GetUserAccount(context.Background(), ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"",
		acc.Settings.Dashboard.DisplayName,
	)
	require.Equal(
		t,
		"acct_123",
		acc.ID,
	)
}
