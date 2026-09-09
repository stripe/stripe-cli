package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestAccountsSignature_orderIndependent(t *testing.T) {
	a := []config.AuthorizedAccount{
		{ID: "acct_1", Name: "One", Modes: []string{"live", "test"}},
		{ID: "acct_2", Name: "Two", Modes: []string{"test"}},
	}
	b := []config.AuthorizedAccount{
		{ID: "acct_2", Name: "Two", Modes: []string{"test"}},
		{ID: "acct_1", Name: "One", Modes: []string{"test", "live"}},
	}
	assert.Equal(t, accountsSignature(a), accountsSignature(b))
}

func TestAccountsSignature_detectsDifference(t *testing.T) {
	a := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test"}}}
	b := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test", "live"}}}
	assert.NotEqual(t, accountsSignature(a), accountsSignature(b))
}

func TestWaitForAccountsChange_detectsChange(t *testing.T) {
	before := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test"}}}
	after := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test", "live"}}}

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := before
		if calls.Add(1) >= 3 {
			resp = after
		}
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: resp}) //nolint:errcheck
	}))
	defer srv.Close()

	got, err := waitForAccountsChange(context.Background(), srv.URL, "oak_test", before, time.Millisecond, time.Second)
	require.NoError(t, err)
	assert.Equal(t, after, got)
	assert.GreaterOrEqual(t, calls.Load(), int32(3))
}

func TestWaitForAccountsChange_timesOut(t *testing.T) {
	before := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: before}) //nolint:errcheck
	}))
	defer srv.Close()

	_, err := waitForAccountsChange(context.Background(), srv.URL, "oak_test", before, time.Millisecond, 20*time.Millisecond)
	assert.ErrorIs(t, err, errReauthTimeout)
}

func TestWaitForAccountsChange_contextCanceled(t *testing.T) {
	before := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listAccountsResponse{Accounts: before}) //nolint:errcheck
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := waitForAccountsChange(ctx, srv.URL, "oak_test", before, time.Millisecond, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWaitForAccountsChange_propagatesRequestError(t *testing.T) {
	before := []config.AuthorizedAccount{{ID: "acct_1", Name: "One", Modes: []string{"test"}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := waitForAccountsChange(context.Background(), srv.URL, "oak_test", before, time.Millisecond, time.Second)
	assert.ErrorContains(t, err, "401")
}
