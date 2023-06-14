package login

import (
	"context"
	"encoding/json"
	"github.com/stripe/stripe-cli/pkg/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/login/acct"
)

const testDisplayName = "test_disp_name"

func TestSuccessMessage(t *testing.T) {
	account := &acct.Account{
		ID: "acct_123",
	}
	account.Settings.Dashboard.DisplayName = testDisplayName

	var apiKey = config.NewAPIKeyFromString("sk_test_123", nil)
	msg, err := SuccessMessage(context.Background(), account, "", apiKey)
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for test_disp_name with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageNoDisplayName(t *testing.T) {
	account := &acct.Account{
		ID: "acct_123",
	}

	var apiKey = config.NewAPIKeyFromString("sk_test_123", nil)
	msg, err := SuccessMessage(context.Background(), account, "", apiKey)
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageBasicMessage(t *testing.T) {
	account := &acct.Account{}

	var apiKey = config.NewAPIKeyFromString("sk_test_123", nil)
	msg, err := SuccessMessage(context.Background(), account, "", apiKey)
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

		account := &acct.Account{
			ID: "acct_123",
		}
		account.Settings.Dashboard.DisplayName = testDisplayName

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	var apiKey = config.NewAPIKeyFromString("sk_test_123", nil)
	msg, err := SuccessMessage(context.Background(), nil, ts.URL, apiKey)
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

		account := &acct.Account{
			ID: "acct_123",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	var apiKey = config.NewAPIKeyFromString("sk_test_123", nil)
	msg, err := SuccessMessage(context.Background(), nil, ts.URL, apiKey)
	require.NoError(t, err)
	require.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}
