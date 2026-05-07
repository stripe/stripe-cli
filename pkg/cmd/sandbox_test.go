package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/sandbox"
)

func setupSandboxTestConfig(t *testing.T) func() {
	t.Helper()
	profilesFile := filepath.Join(t.TempDir(), "config.toml")

	origProfilesFile := Config.ProfilesFile
	origProfileName := Config.Profile.ProfileName

	Config.ProfilesFile = profilesFile
	Config.Profile.ProfileName = "default"
	Config.InitConfig()

	// Mock browser to prevent real browser launches
	origOpen := openBrowserFunc
	origCanOpen := canOpenBrowserFunc
	openBrowserFunc = func(url string) error { return nil }
	canOpenBrowserFunc = func() bool { return true }

	// Mock git config
	origGit := sandbox.GitConfigFunc
	sandbox.GitConfigFunc = func(key string) string { return "" }

	return func() {
		Config.ProfilesFile = origProfilesFile
		Config.Profile.ProfileName = origProfileName
		openBrowserFunc = origOpen
		canOpenBrowserFunc = origCanOpen
		sandbox.GitConfigFunc = origGit
		viper.Reset()
	}
}

func computeChallengeForTest(salt string, number int64) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(strconv.FormatInt(number, 10)))
	return hex.EncodeToString(h.Sum(nil))
}

func sandboxTestServer(t *testing.T, salt string, secretNumber int64, challenge string) *httptest.Server {
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

func dashboardServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				"account_id":               "acct_dash_789",
				"account_display_name":     "Test Corp",
				"testmode_key_secret":      "sk_test_dashboard",
				"testmode_key_publishable": "pk_test_dashboard",
				"livemode_key_secret":      "rk_live_dashboard",
				"livemode_key_publishable": "pk_live_dashboard",
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

func TestSandboxCreateCmd_MissingEmail(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--email is required")
}

func TestSandboxCreateCmd_ProvisionFlow_OutputsJSON(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	salt := "cmd-test-salt"
	secretNumber := int64(5)
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
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sk_test_sandbox", result.SecretKey)
	assert.Equal(t, "pk_test_sandbox", result.PublishableKey)
	assert.Equal(t, "acct_sandbox_123", result.AccountID)
	assert.Contains(t, stderr.String(), "Provisioned!")
	assert.Contains(t, stderr.String(), "Claim your sandbox")
}

func TestSandboxCreateCmd_ProvisionFlow_FallsBackOnServerError(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	// Provision server returns 503 (should trigger fallback)
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "service unavailable")
	}))
	defer failServer.Close()

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "auto", "--base-url", failServer.URL, "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Falling back to browser login")

	var result sandbox.ProvisionResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sk_test_dashboard", result.SecretKey)
}

func TestSandboxCreateCmd_ProvisionFlow_NoFallbackOn400(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "bad@email"
		}
		return ""
	}

	// Server returns 400 (client error — should NOT fallback)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "invalid email")
	}))
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "auto", "--base-url", server.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "400")
	assert.NotContains(t, stderr.String(), "Falling back")
}

func TestSandboxCreateCmd_DashboardFlow(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--dashboard", "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)

	assert.NotContains(t, stderr.String(), "Solving proof-of-work")
	assert.Contains(t, stderr.String(), "Waiting for confirmation")
	assert.Contains(t, stderr.String(), `Connected to "Test Corp"`)

	var result sandbox.ProvisionResponse
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
	assert.Equal(t, "sk_test_dashboard", result.SecretKey)
	assert.Equal(t, "pk_test_dashboard", result.PublishableKey)
	assert.Equal(t, "acct_dash_789", result.AccountID)
}

func TestSandboxCreateCmd_DashboardFlow_DoesNotRequireEmail(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--dashboard", "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.NotContains(t, stderr.String(), "--email is required")
}

func TestSandboxCreateCmd_DashboardFlow_BlockedInSSH(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "/dev/pts/0")

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--dashboard"})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "browser login unavailable in SSH session")
}

func TestSandboxCreateCmd_NoAuthRequired(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	salt := "noauth-salt"
	secretNumber := int64(3)
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
}

func TestSandboxCreateCmd_ConfigNotCorrupted(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	sandbox.GitConfigFunc = func(key string) string {
		if key == "user.email" {
			return "test@stripe.com"
		}
		return ""
	}

	salt := "config-test"
	secretNumber := int64(2)
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

	// Positive: temp config file actually got the key
	content, readErr := os.ReadFile(Config.ProfilesFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "sk_test_sandbox")

	// Negative: real config was not touched
	realConfig := filepath.Join(os.Getenv("HOME"), ".config", "stripe", "config.toml")
	if _, statErr := os.Stat(realConfig); statErr == nil {
		realContent, _ := os.ReadFile(realConfig)
		assert.NotContains(t, string(realContent), "sk_test_sandbox")
	}
}

func TestSaveSandboxToConfig_RejectsInvalidKey(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := saveSandboxToConfig(&sandbox.ProvisionResponse{
		SecretKey:      "garbage",
		PublishableKey: "pk_test_x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid secret key")
}
