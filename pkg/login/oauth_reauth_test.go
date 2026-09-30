package login

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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

func TestPresentReauthURL_immediateOpenDoesNotPrompt(t *testing.T) {
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	origOpenBrowser := openBrowser
	origCanOpenBrowser := canOpenBrowser
	t.Cleanup(func() {
		openBrowser = origOpenBrowser
		canOpenBrowser = origCanOpenBrowser
	})

	const reauthURL = "https://dashboard.stripe.com/reauth"
	var openedURL string
	openBrowser = func(url string) error {
		openedURL = url
		return nil
	}
	canOpenBrowser = func() bool { return true }

	stdout, stderr := captureReauthOutput(t, func() {
		browserOpened := presentReauthURL(reauthURL, true, nil)
		require.NotNil(t, browserOpened)
		select {
		case <-browserOpened:
		default:
			t.Fatal("expected immediate browser handoff to complete before polling")
		}
	})

	assert.Equal(t, reauthURL, openedURL)
	assert.Contains(t, stdout, reauthURL)
	assert.NotContains(t, stdout, "Press enter")
	assert.Empty(t, stderr)
}

func TestPresentReauthURL_withoutBrowserPrintsURL(t *testing.T) {
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	origOpenBrowser := openBrowser
	origCanOpenBrowser := canOpenBrowser
	t.Cleanup(func() {
		openBrowser = origOpenBrowser
		canOpenBrowser = origCanOpenBrowser
	})

	const reauthURL = "https://dashboard.stripe.com/reauth"
	opened := false
	openBrowser = func(string) error {
		opened = true
		return nil
	}
	canOpenBrowser = func() bool { return false }

	stdout, stderr := captureReauthOutput(t, func() {
		browserOpened := presentReauthURL(reauthURL, true, nil)
		assert.Nil(t, browserOpened)
	})

	assert.Contains(t, stdout, reauthURL)
	assert.False(t, opened)
	assert.Empty(t, stderr)
}

func TestPresentReauthURL_browserFailureKeepsManualURL(t *testing.T) {
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	origOpenBrowser := openBrowser
	origCanOpenBrowser := canOpenBrowser
	t.Cleanup(func() {
		openBrowser = origOpenBrowser
		canOpenBrowser = origCanOpenBrowser
	})

	const reauthURL = "https://dashboard.stripe.com/reauth"
	openBrowser = func(string) error { return assert.AnError }
	canOpenBrowser = func() bool { return true }

	stdout, stderr := captureReauthOutput(t, func() {
		browserOpened := presentReauthURL(reauthURL, true, nil)
		require.NotNil(t, browserOpened)
		<-browserOpened
	})

	assert.Contains(t, stdout, reauthURL)
	assert.Contains(t, stderr, "Failed to open browser")
	assert.Contains(t, stderr, assert.AnError.Error())
}

func TestPresentReauthURL_existingModeWaitsForEnter(t *testing.T) {
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	origOpenBrowser := openBrowser
	origCanOpenBrowser := canOpenBrowser
	t.Cleanup(func() {
		openBrowser = origOpenBrowser
		canOpenBrowser = origCanOpenBrowser
	})

	const reauthURL = "https://dashboard.stripe.com/reauth"
	var openedURL string
	openBrowser = func(url string) error {
		openedURL = url
		return nil
	}
	canOpenBrowser = func() bool { return true }

	stdout, stderr := captureReauthOutput(t, func() {
		browserOpened := presentReauthURL(reauthURL, false, func() {})
		require.NotNil(t, browserOpened)
		<-browserOpened
	})

	assert.Contains(t, stdout, "Press enter to open the browser (^C to quit)")
	assert.Equal(t, reauthURL, openedURL)
	assert.Empty(t, stderr)
}

func TestReauthImmediatelyValidatesURLBeforeDisplayingOrOpening(t *testing.T) {
	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	origOpenBrowser := openBrowser
	origCanOpenBrowser := canOpenBrowser
	t.Cleanup(func() {
		openBrowser = origOpenBrowser
		canOpenBrowser = origCanOpenBrowser
	})

	const untrustedURL = "https://evil.example.com/reauth"
	openCalls := 0
	openBrowser = func(string) error {
		openCalls++
		return nil
	}
	canOpenBrowser = func() bool { return true }

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/stripecli/oauth2/token/accounts":
			_, _ = w.Write([]byte(`{"accounts":[]}`))
		case "/stripecli/oauth2/token/reauth":
			_, _ = w.Write([]byte(`{"reauth_url":"` + untrustedURL + `"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var reauthErr error
	stdout, _ := captureReauthOutput(t, func() {
		reauthErr = ReauthImmediately(context.Background(), srv.URL, "oak_test")
	})

	require.Error(t, reauthErr)
	assert.Contains(t, reauthErr.Error(), "untrusted URL")
	assert.NotContains(t, stdout, untrustedURL)
	assert.Equal(t, 0, openCalls)
}

func captureReauthOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	require.NoError(t, err)
	stderrReader, stderrWriter, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	fn()
	require.NoError(t, stdoutWriter.Close())
	require.NoError(t, stderrWriter.Close())

	stdout, err := io.ReadAll(stdoutReader)
	require.NoError(t, err)
	stderr, err := io.ReadAll(stderrReader)
	require.NoError(t, err)
	return string(stdout), string(stderr)
}
