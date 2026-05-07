package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/sandbox"
)

func computeChallengeForTest(salt string, number int) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(strconv.Itoa(number)))
	return hex.EncodeToString(h.Sum(nil))
}

func sandboxTestServer(t *testing.T, salt string, secretNumber int, challenge string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/keys/challenge":
			json.NewEncoder(w).Encode(sandbox.ChallengeResponse{
				Algorithm: "SHA-256",
				Challenge: challenge,
				Salt:      salt,
				Signature: "test-sig",
			})
		case "/keys/provision":
			var req sandbox.ProvisionRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Number != secretNumber {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprint(w, "invalid solution")
				return
			}
			json.NewEncoder(w).Encode(sandbox.ProvisionResponse{
				SecretKey:      "sk_test_sandbox",
				PublishableKey: "pk_test_sandbox",
				ClaimURL:       "https://dashboard.stripe.com/claim_sandbox/test",
				ExpiresAt:      "2026-05-10T00:00:00Z",
				AccountID:      "acct_sandbox_123",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestResolveAutoValue_LiteralEmail(t *testing.T) {
	result, err := resolveAutoValue("user@example.com", "user.email", "--email")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", result)
}

func TestResolveAutoValue_Empty(t *testing.T) {
	_, err := resolveAutoValue("", "user.email", "--email")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email is required")
}

func TestResolveAutoValue_AutoSuccess(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "git@example.com"
		}
		return ""
	}

	result, err := resolveAutoValue("auto", "user.email", "--email")
	require.NoError(t, err)
	assert.Equal(t, "git@example.com", result)
}

func TestResolveAutoValue_AutoMissingGitConfig(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()

	sandbox.GitConfigFunc = func(key string) string { return "" }

	_, err := resolveAutoValue("auto", "user.email", "--email")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git config user.email")
}

func TestResolveAutoValue_AutoName(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.name" {
			return "Jane Smith"
		}
		return ""
	}

	result, err := resolveAutoValue("auto", "user.name", "--name")
	require.NoError(t, err)
	assert.Equal(t, "Jane Smith", result)
}

func TestSandboxCreateCmd_MissingEmail(t *testing.T) {
	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email is required")
}

func TestSandboxCreateCmd_ProvisionFlow_OutputsJSON(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()
	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	salt := "cmd-test-salt"
	secretNumber := 5
	challenge := computeChallengeForTest(salt, secretNumber)

	server := sandboxTestServer(t, salt, secretNumber, challenge)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "auto", "--base-url", server.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)

	var result sandbox.ProvisionResponse
	err = json.Unmarshal(stdout.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "sk_test_sandbox", result.SecretKey)
	assert.Equal(t, "pk_test_sandbox", result.PublishableKey)
	assert.Equal(t, "acct_sandbox_123", result.AccountID)
	assert.Contains(t, stderr.String(), "Provisioned!")
	assert.Contains(t, stderr.String(), "Claim your sandbox")
}

func TestSandboxCreateCmd_ProvisionFlow_FallsBackToDashboard(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()
	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	// Provision server that always fails
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "service unavailable")
	}))
	defer failServer.Close()

	// Dashboard server that succeeds immediately
	dashServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/stripecli/auth":
			pollURL := fmt.Sprintf("http://%s/stripecli/auth/poll-token?secret=s", r.Host)
			json.NewEncoder(w).Encode(map[string]string{
				"browser_url":       "https://dashboard.stripe.com/test",
				"poll_url":          pollURL,
				"verification_code": "code-123",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/stripecli/auth/poll-token":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"redeemed":                 true,
				"account_id":               "acct_fallback_789",
				"account_display_name":     "Fallback Account",
				"testmode_key_secret":      "sk_test_fallback",
				"testmode_key_publishable": "pk_test_fallback",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer dashServer.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "auto", "--base-url", failServer.URL, "--dashboard-base", dashServer.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Provisioning failed")
	assert.Contains(t, stderr.String(), "Falling back to browser login")

	var result sandbox.ProvisionResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sk_test_fallback", result.SecretKey)
	assert.Equal(t, "pk_test_fallback", result.PublishableKey)
	assert.Equal(t, "acct_fallback_789", result.AccountID)
}

func TestSandboxCreateCmd_NoAuthRequired(t *testing.T) {
	original := sandbox.GitConfigFunc
	defer func() { sandbox.GitConfigFunc = original }()
	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	salt := "noauth-salt"
	secretNumber := 3
	challenge := computeChallengeForTest(salt, secretNumber)

	server := sandboxTestServer(t, salt, secretNumber, challenge)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "auto", "--base-url", server.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), "API key")
	assert.NotContains(t, stderr.String(), "login")
}

func TestSandboxCreateCmd_DashboardFlag_SkipsProvision(t *testing.T) {
	dashServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/stripecli/auth":
			pollURL := fmt.Sprintf("http://%s/stripecli/auth/poll-token?secret=s", r.Host)
			json.NewEncoder(w).Encode(map[string]string{
				"browser_url":       "https://dashboard.stripe.com/test",
				"poll_url":          pollURL,
				"verification_code": "code-456",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/stripecli/auth/poll-token":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"redeemed":                 true,
				"account_id":               "acct_dash_direct",
				"testmode_key_secret":      "sk_test_dash_direct",
				"testmode_key_publishable": "pk_test_dash_direct",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer dashServer.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--dashboard", "--dashboard-base", dashServer.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), "Solving proof-of-work")
	assert.Contains(t, stderr.String(), "Waiting for confirmation")

	var result sandbox.ProvisionResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sk_test_dash_direct", result.SecretKey)
}

func TestSandboxCreateCmd_DashboardFlag_DoesNotRequireEmail(t *testing.T) {
	dashServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/stripecli/auth":
			pollURL := fmt.Sprintf("http://%s/stripecli/auth/poll-token?secret=s", r.Host)
			json.NewEncoder(w).Encode(map[string]string{
				"browser_url":       "https://dashboard.stripe.com/test",
				"poll_url":          pollURL,
				"verification_code": "code-789",
			})
		case r.Method == http.MethodGet && r.URL.Path == "/stripecli/auth/poll-token":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"redeemed":                 true,
				"account_id":               "acct_no_email",
				"testmode_key_secret":      "sk_test_no_email",
				"testmode_key_publishable": "pk_test_no_email",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer dashServer.Close()

	// No --email flag, just --dashboard
	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--dashboard", "--dashboard-base", dashServer.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), "--email is required")
}
