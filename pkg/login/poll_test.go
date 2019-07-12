package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestRedeemed(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		response := &pollAPIKeyResponse{
			Redeemed: false,
		}
		if atomic.LoadUint64(&attempts) == 2 {
			response.Redeemed = true
			response.AccountID = "acct_123"
			response.AccountDisplayName = "test_disp_name"
			response.APIKey = "sk_test_123"
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	assert.NoError(t, err)
	assert.Equal(t, "sk_test_123", apiKey)
	assert.Equal(t, "acct_123", account.ID)
	assert.Equal(t, "test_disp_name", account.Settings.Dashboard.DisplayName)
	assert.Equal(t, uint64(2), atomic.LoadUint64(&attempts))
}

func TestRedeemedNoDisplayName(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		atomic.AddUint64(&attempts, 1)

		response := &pollAPIKeyResponse{
			Redeemed: false,
		}
		if atomic.LoadUint64(&attempts) == 2 {
			response.Redeemed = true
			response.AccountID = "acct_123"
			response.APIKey = "sk_test_123"
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	assert.NoError(t, err)
	assert.Equal(t, "sk_test_123", apiKey)
	assert.Equal(t, "acct_123", account.ID)
	assert.Equal(t, "", account.Settings.Dashboard.DisplayName)
	assert.Equal(t, uint64(2), atomic.LoadUint64(&attempts))
}

func TestExceedMaxAttempts(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		response := pollAPIKeyResponse{
			Redeemed: false,
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	assert.EqualError(t, err, "exceeded max attempts")
	assert.Empty(t, apiKey)
	assert.Empty(t, account)
	assert.Equal(t, uint64(3), atomic.LoadUint64(&attempts))
}

func TestHTTPStatusError(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	assert.EqualError(t, err, "unexpected http status code: 500 ")
	assert.Empty(t, apiKey)
	assert.Nil(t, account)
	assert.Equal(t, uint64(1), atomic.LoadUint64(&attempts))
}

func TestHTTPRequestError(t *testing.T) {
	// Immediately close the HTTP server so that the poll request fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connect: connection refused")
	assert.Empty(t, apiKey)
	assert.Nil(t, account)
}
