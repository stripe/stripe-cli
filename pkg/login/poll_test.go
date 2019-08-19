package login

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRedeemed(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

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
	require.NoError(t, err)
	require.Equal(t, "sk_test_123", apiKey)
	require.Equal(t, "acct_123", account.ID)
	require.Equal(t, "test_disp_name", account.Settings.Dashboard.DisplayName)
	require.Equal(t, uint64(2), atomic.LoadUint64(&attempts))
}

func TestRedeemedNoDisplayName(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)

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
	require.NoError(t, err)
	require.Equal(t, "sk_test_123", apiKey)
	require.Equal(t, "acct_123", account.ID)
	require.Equal(t, "", account.Settings.Dashboard.DisplayName)
	require.Equal(t, uint64(2), atomic.LoadUint64(&attempts))
}

func TestExceedMaxAttempts(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

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
	require.EqualError(t, err, "exceeded max attempts")
	require.Empty(t, apiKey)
	require.Empty(t, account)
	require.Equal(t, uint64(3), atomic.LoadUint64(&attempts))
}

func TestHTTPStatusError(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	require.EqualError(t, err, "unexpected http status code: 500 ")
	require.Empty(t, apiKey)
	require.Nil(t, account)
	require.Equal(t, uint64(1), atomic.LoadUint64(&attempts))
}

func TestHTTPRequestError(t *testing.T) {
	// Immediately close the HTTP server so that the poll request fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	apiKey, account, err := PollForKey(ts.URL, 1*time.Millisecond, 3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "connect: connection refused")
	require.Empty(t, apiKey)
	require.Nil(t, account)
}
