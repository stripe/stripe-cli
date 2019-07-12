package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSuccessMessage(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}
	account.Settings.Dashboard.DisplayName = "test_disp_name"

	msg := SuccessMessage(account, "", "sk_test_123")
	assert.Equal(
		t,
		"Done! The Stripe CLI is configured for test_disp_name with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageNoDisplayName(t *testing.T) {
	account := &Account{
		ID: "acct_123",
	}

	msg := SuccessMessage(account, "", "sk_test_123")
	assert.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageBasicMessage(t *testing.T) {
	account := &Account{}
	msg := SuccessMessage(account, "", "sk_test_123")
	assert.Equal(
		t,
		"Done! The Stripe CLI is configured\n",
		msg,
	)
}

func TestSuccessMessageGetAccount(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}
		account.Settings.Dashboard.DisplayName = "test_disp_name"

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	msg := SuccessMessage(nil, ts.URL, "sk_test_123")
	assert.Equal(
		t,
		"Done! The Stripe CLI is configured for test_disp_name with account id acct_123\n",
		msg,
	)
}

func TestSuccessMessageGetAccountNoDisplayName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		account := &Account{
			ID: "acct_123",
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(account)
	}))
	defer ts.Close()

	msg := SuccessMessage(nil, ts.URL, "sk_test_123")
	assert.Equal(
		t,
		"Done! The Stripe CLI is configured for your account with account id acct_123\n",
		msg,
	)
}
