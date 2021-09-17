package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const testDisplayName = "test_disp_name"

func TestSuccessMessage(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}
	account.Settings.Dashboard.DisplayName = testDisplayName

	msg, err := SuccessMessage(context.Background(), account, "", "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for test_disp_name with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageNoDisplayName(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}

	msg, err := SuccessMessage(context.Background(), account, "", "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageBasicMessage(t *testing.T) {
	account := &Account{}
	msg, err := SuccessMessage(context.Background(), account, "", "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured\n",
		msg,
	)
}

func TestSuccessMessageGetAccount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}
		account.Settings.Dashboard.DisplayName = testDisplayName

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	msg, err := SuccessMessage(context.Background(), nil, ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for test_disp_name with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageGetAccountNoDisplayName(t *testing.T) {
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

	msg, err := SuccessMessage(context.Background(), nil, ts.URL, "sk_test_123")
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}
