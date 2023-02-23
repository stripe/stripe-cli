package keys

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/login/acct"
)

type mockConfigurer struct {
	called bool
}

func (c *mockConfigurer) SaveLoginDetails(response *PollAPIKeyResponse) error {
	c.called = true
	return nil
}

func TestAsyncPollKey_Succeeds(t *testing.T) {
	var attempts uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		response := &PollAPIKeyResponse{
			Redeemed: false,
		}
		if atomic.LoadUint64(&attempts) == 2 {
			response.Redeemed = true
			response.AccountID = "acct_123"
			response.AccountDisplayName = "test_disp_name"
			response.TestModeAPIKey = "sk_test_123"
			response.TestModePublishableKey = "pk_test_123"
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	ch := make(chan AsyncPollResult)
	configurer := mockConfigurer{}
	rt := NewRAKTransfer(&configurer)

	go rt.AsyncPollKey(context.Background(), ts.URL, 1*time.Millisecond, 3, ch)

	result := <-ch

	require.True(t, configurer.called, "expected SaveLoginDetails to be called, but wasn't")
	require.NoError(t, result.Err)
	assert.Equal(t, "sk_test_123", result.TestModeAPIKey)
	assert.Equal(t, &acct.Account{
		ID: "acct_123",
		Settings: acct.Settings{
			Dashboard: acct.Dashboard{
				DisplayName: "test_disp_name",
			},
		},
	}, result.Account)
}

func TestAsyncPollKey_ResponseError(t *testing.T) {
	var attempts uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	ch := make(chan AsyncPollResult)
	configurer := mockConfigurer{}
	rt := NewRAKTransfer(&configurer)

	go rt.AsyncPollKey(context.Background(), ts.URL, 1*time.Millisecond, 3, ch)

	result := <-ch
	require.False(t, configurer.called, "SaveLoginDetails was unexpectedly called")
	assert.EqualError(t, result.Err, "unexpected http status code: 500 ")
}

func TestAsyncPollKey_ExceedMaxAttempts(t *testing.T) {
	var attempts uint64

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		response := PollAPIKeyResponse{
			Redeemed: false,
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	ch := make(chan AsyncPollResult)
	configurer := mockConfigurer{}
	rt := NewRAKTransfer(&configurer)

	go rt.AsyncPollKey(context.Background(), ts.URL, 1*time.Millisecond, 3, ch)

	result := <-ch
	require.False(t, configurer.called, "SaveLoginDetails was unexpectedly called")
	assert.EqualError(t, result.Err, "exceeded max attempts")
	assert.Equal(t, uint64(3), atomic.LoadUint64(&attempts))
}

type errorConfigurer struct {
	called bool
}

func (c *errorConfigurer) SaveLoginDetails(response *PollAPIKeyResponse) error {
	c.called = true
	return errors.New("failed to save login details")
}

func TestAsyncPollKey_ConfigurerError(t *testing.T) {
	var attempts uint64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)

		atomic.AddUint64(&attempts, 1)

		response := &PollAPIKeyResponse{
			Redeemed: false,
		}
		if atomic.LoadUint64(&attempts) == 2 {
			response.Redeemed = true
			response.AccountID = "acct_123"
			response.AccountDisplayName = "test_disp_name"
			response.TestModeAPIKey = "sk_test_123"
			response.TestModePublishableKey = "pk_test_123"
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	ch := make(chan AsyncPollResult)
	configurer := errorConfigurer{}
	rt := NewRAKTransfer(&configurer)

	go rt.AsyncPollKey(context.Background(), ts.URL, 1*time.Millisecond, 3, ch)

	result := <-ch
	require.True(t, configurer.called, "expected SaveLoginDetails to be called, but wasn't")
	assert.EqualError(t, result.Err, "failed to save login details")
}
