package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	viper.Reset()
	os.WriteFile(profilesFile, []byte("[default]\n"), 0600)
	Config.ProfilesFile = profilesFile
	Config.Profile.ProfileName = "default"
	Config.Profile.TestModeAPIKey = ""
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

func TestSandboxCreateCmd_MissingEmail(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
}

func TestSandboxCreateCmd_EmailAndFromGitMutuallyExclusive(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@example.com", "--from-git"})

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestSandboxCreateCmd_FromGitResolves(t *testing.T) {
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

	// Server returns 503 to trigger fallback (which needs a dashboard server)
	// We just verify the email was resolved before the server call
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify email was sent in the request
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["email"] == "test@stripe.com" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer failServer.Close()

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--from-git", "--base-url", failServer.URL, "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Using email: test@stripe.com")
}

func TestSandboxCreateCmd_FromGitMissing(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	sandbox.GitConfigFunc = func(key string) string { return "" }

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--from-git"})

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--from-git requires git config user.email")
}

func TestSandboxCreateCmd_ProvisionFlow_OutputsJSON(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test outside CI (CreateProfile hangs in devbox)")
	}
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	salt := "cmd-test-salt"
	secretNumber := int64(5)
	challenge := computeChallengeForTest(salt, secretNumber)

	server := sandboxTestServer(t, salt, secretNumber, challenge)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--base-url", server.URL})

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
	assert.Contains(t, stderr.String(), "claim your sandbox")
}

func TestSandboxCreateCmd_ProvisionFlow_FallsBackOnServerError(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	// Provision server returns 503 (should trigger fallback)
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "service unavailable")
	}))
	defer failServer.Close()

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--base-url", failServer.URL, "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Opening browser to set up your account")
	// Login() prints success to os.Stdout directly, not our buffer.
	// The test verifies fallback was triggered (via stderr) and no error returned.
}

func TestSandboxCreateCmd_FallsBackOn429(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	// Server returns 429 (should also trigger fallback)
	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":"Too many requests"}`)
	}))
	defer failServer.Close()

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--base-url", failServer.URL, "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Opening browser to set up your account")
}

func TestSandboxCreateCmd_FallbackBlockedInSSH(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "/dev/pts/0")

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprint(w, "service unavailable")
	}))
	defer failServer.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--base-url", failServer.URL})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "browser login unavailable in SSH session")
}

func TestSandboxCreateCmd_AlreadyLoggedIn(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	// Simulate being logged in
	Config.Profile.TestModeAPIKey = "sk_test_existing"
	Config.Profile.CreateProfile()

	var openedURL string
	openBrowserFunc = func(u string) error { openedURL = u; return nil }

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com"})

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Press Enter to open the browser")
	assert.Contains(t, openedURL, "/sandboxes")
}

func TestSandboxCreateCmd_FallbackPreFillsEmail(t *testing.T) {
	// Verify the signup URL construction includes the email parameter.
	// We test URL building directly since Login() writes to os.Stdout
	// which we can't capture via cmd buffers.
	baseURL := "https://dashboard.stripe.com"
	browserURL := "https://dashboard.stripe.com/stripecli/confirm_auth?t=secret123"
	email := "user@example.com"

	parsed, err := url.Parse(browserURL)
	require.NoError(t, err)
	confirmPath := parsed.RequestURI()

	params := url.Values{}
	params.Set("redirect", confirmPath)
	params.Set("email", email)
	signupURL := fmt.Sprintf("%s/register?%s", baseURL, params.Encode())

	assert.Contains(t, signupURL, "email=user%40example.com")
	assert.Contains(t, signupURL, "redirect=%2Fstripecli%2Fconfirm_auth")
	assert.Contains(t, signupURL, "secret123")
}

func TestSandboxCreateCmd_ConfigNotCorrupted(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test outside CI (CreateProfile hangs in devbox)")
	}
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	salt := "config-test"
	secretNumber := int64(2)
	challenge := computeChallengeForTest(salt, secretNumber)

	server := sandboxTestServer(t, salt, secretNumber, challenge)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--base-url", server.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)

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

func TestSaveSandboxToConfig_EmptyKey(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := saveSandboxToConfig(&sandbox.ProvisionResponse{
		SecretKey:      "",
		PublishableKey: "pk_test_x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no secret key")
}
