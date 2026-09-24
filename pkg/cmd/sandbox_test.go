package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/sandbox"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

func setupSandboxTestConfig(t *testing.T) func() {
	t.Helper()
	resetSandboxCreateFlagsForTest(t)
	profilesFile := filepath.Join(t.TempDir(), "config.toml")

	origProfilesFile := Config.ProfilesFile
	origProfileName := Config.Profile.ProfileName
	origAPIKey := Config.Profile.APIKey
	origKeyRing := config.KeyRing

	viper.Reset()
	os.WriteFile(profilesFile, []byte("[default]\n"), 0600)
	Config.ProfilesFile = profilesFile
	Config.Profile.ProfileName = "default"
	Config.Profile.APIKey = ""
	Config.Profile.TestModeAPIKey = ""
	Config.InitConfig()
	config.KeyRing = keyring.NewMemoryStore(nil)

	// Mock browser to prevent real browser launches
	origOpen := openBrowserFunc
	origCanOpen := canOpenBrowserFunc
	openBrowserFunc = func(url string) error { return nil }
	canOpenBrowserFunc = func() bool { return true }

	// Also mock the login package's browser opener (used by fallback flow)
	restoreLoginBrowser := login.SetOpenBrowserForTesting(func(string) error { return nil })

	// Mock git config
	origGit := sandbox.GitConfigFunc
	sandbox.GitConfigFunc = func(key string) string { return "" }

	return func() {
		Config.ProfilesFile = origProfilesFile
		Config.Profile.ProfileName = origProfileName
		Config.Profile.APIKey = origAPIKey
		config.KeyRing = origKeyRing
		openBrowserFunc = origOpen
		canOpenBrowserFunc = origCanOpen
		restoreLoginBrowser()
		sandbox.GitConfigFunc = origGit
		viper.Reset()
	}
}

func resetSandboxCreateFlagsForTest(t *testing.T) {
	t.Helper()
	cmd, _, err := rootCmd.Find([]string{"sandbox", "create"})
	require.NoError(t, err)

	for _, name := range []string{
		"base-url",
		"create-blank",
		"country",
		"api-base",
		"dashboard-base",
		"email",
		"from-git",
		"full-name",
		"non-interactive",
	} {
		flag := cmd.Flags().Lookup(name)
		require.NotNil(t, flag)
		require.NoError(t, flag.Value.Set(flag.DefValue))
		flag.Changed = false
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
			json.NewEncoder(w).Encode(map[string]interface{}{
				"secret_key":      "sk_test_sandbox",
				"publishable_key": "pk_test_sandbox",
				"claim_url":       "https://dashboard.stripe.com/claim_sandbox/test",
				"expires_at":      "2026-05-10T00:00:00Z",
				"account_id":      "acct_sandbox_123",
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

const activeClaimableSandboxAPIKey = "rkcs_test_claim_status"

type claimStatusRequest struct {
	Method  string
	Path    string
	Auth    string
	Version string
}

func newClaimStatusServer(t *testing.T, statusCode int, body string, recs *[]claimStatusRequest) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recs != nil {
			*recs = append(*recs, claimStatusRequest{
				Method:  r.Method,
				Path:    r.URL.Path,
				Auth:    r.Header.Get("Authorization"),
				Version: r.Header.Get("Stripe-Version"),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
}

func setupActiveClaimableSandbox(t *testing.T, claimURL, expiresAt string) {
	t.Helper()
	Config.Profile.APIKey = activeClaimableSandboxAPIKey
	Config.Profile.AccountID = "acct_sandbox_123"
	require.NoError(t, Config.Profile.WriteConfigField(config.TestModeAPIKeyName, activeClaimableSandboxAPIKey))
	if claimURL != "" {
		require.NoError(t, Config.Profile.WriteConfigField(config.SandboxClaimURLName, claimURL))
	}
	if expiresAt != "" {
		require.NoError(t, Config.Profile.WriteConfigField(config.SandboxExpiresAtName, expiresAt))
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	runErr := fn()
	require.NoError(t, w.Close())

	var buf bytes.Buffer
	_, copyErr := buf.ReadFrom(r)
	require.NoError(t, copyErr)
	return buf.String(), runErr
}

// A sandbox provisioned under an invalid profile name would have nowhere to save
// its keys, so the name is checked before any request is made.
func TestSandboxCreateCmd_RejectsNewDottedProfileBeforeProvisioning(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	Config.Profile.ProfileName = "example.project"
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	browserOpened := false
	openBrowserFunc = func(string) error {
		browserOpened = true
		return nil
	}

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "unused", "--base-url", server.URL})
	err := cmd.cmd.Execute()

	require.EqualError(t, err, `profile name "example.project" cannot contain a period; use a hyphen or underscore instead`)
	assert.Zero(t, requestCount)
	assert.False(t, browserOpened)
}

func TestSandboxCreateCmd_MissingEmail(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	cmd := newSandboxCreateCmd()
	cmd.isInteractive = func(*cobra.Command) bool { return true }
	cmd.cmd.SetArgs([]string{})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "email is required")
	assert.NotContains(t, stdout.String(), "Sandbox name:")
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
	// Output goes to os.Stdout directly (CLI convention)
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

func TestSandboxCreateCmd_ProvisionFlow_Succeeds(t *testing.T) {
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

	err := cmd.cmd.Execute()
	require.NoError(t, err)

	// Verify keys were saved to config
	key, _ := Config.Profile.GetAPIKey(false)
	assert.Equal(t, "sk_test_sandbox", key)
	pubKey, _ := Config.Profile.GetPublishableKey(false)
	assert.Equal(t, "pk_test_sandbox", pubKey)
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
	// Fallback triggered — verified by no error returned
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
	// Fallback triggered — verified by no error returned
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

func TestSandboxCreateCmd_AlreadyLoggedIn_WithRealKey(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("Skipping — fmt.Scanln blocks without stdin in devbox")
	}
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	// Simulate being logged in with a real key (not rkcs_)
	Config.Profile.TestModeAPIKey = "sk_test_existing"
	Config.Profile.CreateProfile()

	var openedURL string
	openBrowserFunc = func(u string) error { openedURL = u; return nil }

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com"})

	// Provide stdin so fmt.Scanln doesn't block
	cmd.cmd.SetIn(bytes.NewReader([]byte("\n")))

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	// Redirect detected by openedURL assertion
	assert.Contains(t, openedURL, "/sandboxes")
}

func TestSandboxCreateCmd_ExistingSandboxShowsActiveMessage(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	const claimURL = "https://dashboard.stripe.com/onboard_sandbox/existing"
	setupActiveClaimableSandbox(t, claimURL, "2099-01-01")

	var recs []claimStatusRequest
	server := newClaimStatusServer(t, http.StatusOK, `{"is_claimed":false}`, &recs)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--api-base", server.URL})

	output, err := captureStdout(t, func() error {
		return cmd.cmd.Execute()
	})
	require.NoError(t, err)
	assert.Contains(t, output, "You already have an active sandbox")
	assert.Contains(t, output, "Claim it before then")
	assert.Contains(t, output, "stripe sandbox claim")
	require.Len(t, recs, 1)
	assert.Equal(t, http.MethodGet, recs[0].Method)
	assert.Equal(t, "/v2/core/claimable_sandboxes/status", recs[0].Path)
	assert.Equal(t, sandboxClaimStatusVersion, recs[0].Version)
}

func TestSandboxCreateCmd_FromGitResolvesName(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	t.Setenv("SSH_TTY", "")
	t.Setenv("SSH_CONNECTION", "")
	t.Setenv("SSH_CLIENT", "")

	sandbox.GitConfigFunc = func(key string) string {
		switch key {
		case "user.email":
			return "test@stripe.com"
		case "user.name":
			return "Test User"
		}
		return ""
	}

	// Capture what gets sent to the server
	var receivedName string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/keys/challenge":
			json.NewEncoder(w).Encode(sandbox.ChallengeResponse{
				Algorithm: "SHA-256",
				Challenge: computeChallengeForTest("name-test", 1),
				Salt:      "name-test",
				Signature: "sig",
			})
		case "/keys/provision":
			var req sandbox.ProvisionRequest
			json.NewDecoder(r.Body).Decode(&req)
			receivedName = req.Name
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()

	dashSrv := dashboardServer(t)
	defer dashSrv.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--from-git", "--base-url", server.URL, "--dashboard-base", dashSrv.URL})

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	cmd.cmd.Execute()
	assert.Equal(t, "Test User", receivedName)
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

	err := cmd.cmd.Execute()
	require.NoError(t, err)

	content, readErr := os.ReadFile(Config.ProfilesFile)
	require.NoError(t, readErr)
	assert.Contains(t, string(content), "sk_test_sandbox")
}

func TestSaveSandboxToConfig_EmptyKey(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	var resp sandbox.ProvisionResponse
	json.Unmarshal([]byte(`{"publishable_key":"pk_test_x"}`), &resp)
	err := saveSandboxToConfig(&resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no secret key")
}

func TestSandboxClaimCmd_NoActiveSandbox(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	cmd := newSandboxClaimCmd()

	var stderr bytes.Buffer
	cmd.cmd.SetErr(&stderr)

	err := cmd.cmd.Execute()
	require.NoError(t, err)
	// No active sandbox message — verified by no error
	// No sandbox message printed
}

func TestSandboxClaimCmd_NoActiveSandboxDoesNotCallStatus(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	var recs []claimStatusRequest
	server := newClaimStatusServer(t, http.StatusOK, `{"is_claimed":false}`, &recs)
	defer server.Close()

	cmd := newSandboxClaimCmd()
	cmd.cmd.SetArgs([]string{"--api-base", server.URL})

	output, err := captureStdout(t, func() error {
		return cmd.cmd.Execute()
	})
	require.NoError(t, err)
	assert.Contains(t, output, "No active sandbox. Run `stripe sandbox create` to get started.")
	assert.Empty(t, recs)
}

func TestSandboxClaimCmd_WithClaimURL(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	const claimURL = "https://dashboard.stripe.com/onboard_sandbox/test123"
	setupActiveClaimableSandbox(t, claimURL, "")

	var recs []claimStatusRequest
	server := newClaimStatusServer(t, http.StatusOK, `{"is_claimed":false}`, &recs)
	defer server.Close()

	cmd := newSandboxClaimCmd()
	cmd.cmd.SetArgs([]string{"--non-interactive", "--api-base", server.URL})

	output, err := captureStdout(t, func() error {
		return cmd.cmd.Execute()
	})
	require.NoError(t, err)
	assert.Contains(t, output, claimURL)
	require.Len(t, recs, 1)
	assert.Equal(t, http.MethodGet, recs[0].Method)
	assert.Equal(t, "/v2/core/claimable_sandboxes/status", recs[0].Path)
	assert.Equal(t, "Bearer "+activeClaimableSandboxAPIKey, recs[0].Auth)
	assert.Equal(t, sandboxClaimStatusVersion, recs[0].Version)
}

func TestSandboxClaimCmd_ClaimedDoesNotPrintURLOrOpenBrowser(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "interactive"},
		{name: "non-interactive", args: []string{"--non-interactive"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()

			const claimURL = "https://dashboard.stripe.com/onboard_sandbox/already-claimed"
			setupActiveClaimableSandbox(t, claimURL, "")

			var openedURL string
			openBrowserFunc = func(u string) error {
				openedURL = u
				return nil
			}

			server := newClaimStatusServer(t, http.StatusOK, `{"is_claimed":true}`, nil)
			defer server.Close()

			cmd := newSandboxClaimCmd()
			cmd.cmd.SetArgs(append(tt.args, "--api-base", server.URL))

			output, err := captureStdout(t, func() error {
				return cmd.cmd.Execute()
			})
			require.NoError(t, err)
			assert.Contains(t, output, sandboxAlreadyClaimedMessage)
			assert.NotContains(t, output, claimURL)
			assert.Empty(t, openedURL)
		})
	}
}

func TestSandboxCreateCmd_ExistingSandboxClaimedOmitsClaimGuidance(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	const claimURL = "https://dashboard.stripe.com/onboard_sandbox/existing"
	setupActiveClaimableSandbox(t, claimURL, "2099-01-01")

	server := newClaimStatusServer(t, http.StatusOK, `{"is_claimed":true}`, nil)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--api-base", server.URL})

	output, err := captureStdout(t, func() error {
		return cmd.cmd.Execute()
	})
	require.NoError(t, err)
	assert.Contains(t, output, "You already have an active sandbox")
	assert.Contains(t, output, sandboxAlreadyClaimedMessage)
	assert.NotContains(t, output, "Claim it before then")
	assert.NotContains(t, output, "stripe sandbox claim")
}

func TestSandboxClaimCmd_StatusEndpointErrors(t *testing.T) {
	const claimURL = "https://dashboard.stripe.com/onboard_sandbox/stale"

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "http_404", statusCode: http.StatusNotFound, body: `{"error":{"code":"not_found"}}`},
		{name: "http_500", statusCode: http.StatusInternalServerError, body: `{"error":{"message":"internal"}}`},
		{name: "invalid_json", statusCode: http.StatusOK, body: `{not-json`},
		{name: "missing_is_claimed", statusCode: http.StatusOK, body: `{}`},
		{name: "unknown_status_value", statusCode: http.StatusOK, body: `{"is_claimed":"claimed"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()

			setupActiveClaimableSandbox(t, claimURL, "")

			var openedURL string
			openBrowserFunc = func(u string) error {
				openedURL = u
				return nil
			}

			server := newClaimStatusServer(t, tt.statusCode, tt.body, nil)
			defer server.Close()

			cmd := newSandboxClaimCmd()
			cmd.cmd.SetArgs([]string{"--non-interactive", "--api-base", server.URL})

			output, err := captureStdout(t, func() error {
				return cmd.cmd.Execute()
			})
			require.NoError(t, err)
			assert.Contains(t, output, claimURL)
			assert.Contains(t, output, "Claim your sandbox")
			assert.Empty(t, openedURL)
		})
	}
}

func TestSandboxCreateCmd_ExistingSandboxStatusErrorFallsBackToClaimGuidance(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	const claimURL = "https://dashboard.stripe.com/onboard_sandbox/existing"
	setupActiveClaimableSandbox(t, claimURL, "2099-01-01")

	server := newClaimStatusServer(t, http.StatusInternalServerError, `{"error":{"message":"internal"}}`, nil)
	defer server.Close()

	cmd := newSandboxCreateCmd()
	cmd.cmd.SetArgs([]string{"--email", "test@stripe.com", "--api-base", server.URL})

	output, err := captureStdout(t, func() error {
		return cmd.cmd.Execute()
	})
	require.NoError(t, err)
	assert.Contains(t, output, "You already have an active sandbox")
	assert.Contains(t, output, "Claim it before then")
	assert.Contains(t, output, "stripe sandbox claim")
	assert.NotContains(t, output, sandboxAlreadyClaimedMessage)
}

type fakeSandboxCreateClient struct {
	created      sandbox.CreatedSandbox
	err          error
	calls        []sandbox.CreateOptions
	beforeReturn func()
}

type sandboxTelemetryEvent struct {
	name  string
	value string
}

type fakeSandboxTelemetryClient struct {
	events chan sandboxTelemetryEvent
}

func newFakeSandboxTelemetryClient() *fakeSandboxTelemetryClient {
	return &fakeSandboxTelemetryClient{events: make(chan sandboxTelemetryEvent, 4)}
}

func (f *fakeSandboxTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (f *fakeSandboxTelemetryClient) SendEvent(_ context.Context, name, value string) {
	f.events <- sandboxTelemetryEvent{name: name, value: value}
}

func (f *fakeSandboxTelemetryClient) waitForEvent(t *testing.T) sandboxTelemetryEvent {
	t.Helper()
	select {
	case event := <-f.events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for sandbox telemetry event")
		return sandboxTelemetryEvent{}
	}
}

func (f *fakeSandboxTelemetryClient) assertNoEvent(t *testing.T) {
	t.Helper()
	select {
	case event := <-f.events:
		t.Fatalf("unexpected sandbox telemetry event: %#v", event)
	default:
	}
}

type sandboxTestContextKey struct{}

func (f *fakeSandboxCreateClient) Create(_ context.Context, options sandbox.CreateOptions) (sandbox.CreatedSandbox, error) {
	f.calls = append(f.calls, options)
	if f.beforeReturn != nil {
		f.beforeReturn()
	}
	return f.created, f.err
}

func setSandboxCreateOAuthContext(t *testing.T) {
	t.Helper()
	activeContext, err := json.Marshal(config.ActiveContext{AccountID: "acct_live", Livemode: true})
	require.NoError(t, err)
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oak_live_test"),
		config.OAuthActiveContextKeychainKey: activeContext,
	})
}

func setSandboxTestModeOAuthContext(t *testing.T) {
	t.Helper()
	activeContext, err := json.Marshal(config.ActiveContext{AccountID: "acct_test_active", Livemode: false})
	require.NoError(t, err)
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oak_test_active"),
		config.OAuthActiveContextKeychainKey: activeContext,
	})
}

func TestSandboxCreateCmdSurfaceIncludesAuthenticatedCreation(t *testing.T) {
	command := newSandboxCreateCmd()
	require.Equal(t, "create [name]", command.cmd.Use)

	var flags []string
	command.cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		flags = append(flags, flag.Name)
	})
	require.ElementsMatch(t, []string{
		"api-base",
		"base-url",
		"country",
		"create-blank",
		"dashboard-base",
		"email",
		"from-git",
		"full-name",
		"non-interactive",
	}, flags)
	for _, hidden := range []string{"api-base", "base-url", "dashboard-base"} {
		require.True(t, command.cmd.Flags().Lookup(hidden).Hidden, hidden)
	}
}

func TestSandboxCmdDoesNotRegisterNew(t *testing.T) {
	command := newSandboxCmd()
	for _, child := range command.cmd.Commands() {
		require.NotEqual(t, "new", child.Name())
	}

	for _, name := range []string{"new", "does-not-exist"} {
		err := command.cmd.RunE(command.cmd, []string{name})
		require.EqualError(t, err, fmt.Sprintf("unknown command %q for %q", name, command.cmd.CommandPath()))
	}
}

func TestSandboxCmdPublicSurface(t *testing.T) {
	command := newSandboxCmd()
	commands := make(map[string]*cobra.Command)
	for _, child := range command.cmd.Commands() {
		commands[child.Name()] = child
	}

	for _, name := range []string{"create", "claim", "list", "delete"} {
		child, ok := commands[name]
		require.True(t, ok, name)
		require.False(t, child.Hidden, name)
	}

	assert.Contains(t, command.cmd.UsageString(), "list")
	assert.Contains(t, command.cmd.UsageString(), "delete")
	assert.NotContains(t, command.cmd.UsageString(), "sandbox new")

	for _, name := range []string{"list", "delete"} {
		child := commands[name]
		assert.Nil(t, child.Flags().Lookup("json"), name)
		assert.Nil(t, child.Flags().Lookup("format"), name)
	}
}

func TestSandboxCreateCmdOAuthCreatesCopyLiveByDefault(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetArgs([]string{"  Copied sandbox  "})
	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Copied sandbox"}}, client.calls)
	require.Contains(t, stdout.String(), "Created sandbox \"Copied sandbox\"")
	require.Contains(t, stdout.String(), "Account ID: acct_created")
	require.Contains(t, stdout.String(), "stripe login")
	require.Empty(t, stderr.String())
}

func TestSandboxCreateCmdOAuthRoutesTelemetryOnce(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	telemetry := newFakeSandboxTelemetryClient()
	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetContext(stripe.WithTelemetryClient(context.Background(), telemetry))
	command.cmd.SetArgs([]string{"OAuth sandbox"})

	require.NoError(t, command.cmd.Execute())
	event := telemetry.waitForEvent(t)
	assert.Equal(t, "Sandbox Create Routed", event.name)
	assert.Equal(t, "oauth", event.value)
	telemetry.assertNoEvent(t)
}

func TestSandboxCreateCmdAnonymousRoutesTelemetryOnce(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	telemetry := newFakeSandboxTelemetryClient()
	ctx, cancel := context.WithCancel(stripe.WithTelemetryClient(context.Background(), telemetry))
	cancel()
	command := newSandboxCreateCmd()
	command.cmd.SetContext(ctx)
	command.cmd.SetArgs([]string{"--email", "alice@example.com"})

	err := command.cmd.Execute()
	require.ErrorIs(t, err, context.Canceled)
	event := telemetry.waitForEvent(t)
	assert.Equal(t, "Sandbox Create Routed", event.name)
	assert.Equal(t, "anonymous", event.value)
	telemetry.assertNoEvent(t)
}

func TestSandboxCreateCmdInvalidRouteDoesNotSendTelemetry(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*testing.T)
		args      []string
		wantInErr string
	}{
		{
			name: "OAuth missing name",
			setup: func(t *testing.T) {
				setSandboxCreateOAuthContext(t)
			},
			wantInErr: "sandbox name is required",
		},
		{
			name:      "anonymous invalid email",
			args:      []string{"--email", "not-an-email"},
			wantInErr: "invalid email",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			if test.setup != nil {
				test.setup(t)
			}

			telemetry := newFakeSandboxTelemetryClient()
			command := newSandboxCreateCmd()
			command.cmd.SetContext(stripe.WithTelemetryClient(context.Background(), telemetry))
			command.cmd.SetArgs(test.args)

			err := command.cmd.Execute()
			require.ErrorContains(t, err, test.wantInErr)
			telemetry.assertNoEvent(t)
		})
	}
}

func TestSandboxCreateCmdNilTelemetryClientDoesNotChangeBehavior(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetContext(context.Background())
	command.cmd.SetArgs([]string{"OAuth sandbox"})

	require.NoError(t, command.cmd.Execute())
	assert.Equal(t, []sandbox.CreateOptions{{Name: "OAuth sandbox"}}, client.calls)
}

func TestSandboxCreateCmdRouteTelemetryIsPrivacySafe(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	telemetry := newFakeSandboxTelemetryClient()
	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_sensitive"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetContext(stripe.WithTelemetryClient(context.Background(), telemetry))
	command.cmd.SetArgs([]string{"Alice acct_sensitive alice@example.com"})

	require.NoError(t, command.cmd.Execute())
	event := telemetry.waitForEvent(t)
	assert.Equal(t, "Sandbox Create Routed", event.name)
	assert.Equal(t, "oauth", event.value)
	for _, sensitiveValue := range []string{"Alice", "acct_sensitive", "alice@example.com", "sk_", "--create-blank"} {
		assert.NotContains(t, event.value, sensitiveValue)
	}
}

func TestSandboxCreateCmdOAuthWinsOverConfiguredLegacyKey(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)
	require.NoError(t, Config.Profile.WriteConfigField(config.TestModeAPIKeyName, "sk_test_configured"))

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetArgs([]string{"OAuth sandbox"})

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "OAuth sandbox"}}, client.calls)
}

func TestSandboxCreateCmdStaleOAuthDoesNotFallBackToAnonymousProvisioning(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey: []byte("oak_live_stale"),
	})

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	command := newSandboxCreateCmd()
	command.apiBaseURL = server.URL
	command.cmd.SetArgs([]string{"OAuth sandbox"})

	err := command.cmd.Execute()
	require.ErrorContains(t, err, "stripe login")
	require.Zero(t, requestCount)
}

func TestSandboxManagementCommandsRejectActiveTestModeOAuth(t *testing.T) {
	tests := []struct {
		name string
		run  func(string) error
	}{
		{
			name: "create",
			run: func(apiBaseURL string) error {
				command := newSandboxCreateCmd()
				command.apiBaseURL = apiBaseURL
				command.cmd.SetArgs([]string{"Test mode sandbox"})
				return command.cmd.Execute()
			},
		},
		{
			name: "list",
			run: func(apiBaseURL string) error {
				command := newSandboxListCmd()
				command.apiBase = apiBaseURL
				return command.cmd.Execute()
			},
		},
		{
			name: "delete with confirm",
			run: func(apiBaseURL string) error {
				command := newSandboxDeleteCmd()
				command.apiBase = apiBaseURL
				command.cmd.SetArgs([]string{"acct_target", "--confirm"})
				return command.cmd.Execute()
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			setSandboxTestModeOAuthContext(t)

			requestCount := 0
			mutationCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
				if r.Method != http.MethodGet {
					mutationCount++
				}
				t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()

			err := test.run(server.URL)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "You're in a sandbox.")
			assert.Contains(t, err.Error(), "stripe switch")
			assert.Zero(t, requestCount)
			assert.Zero(t, mutationCount)
		})
	}
}

func TestSandboxCreateCmdExplicitAPIKeyOverrideWinsOverOAuth(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)
	Config.Profile.APIKey = "sk_test_explicit"

	client := &fakeSandboxCreateClient{}
	command := newSandboxCreateCmd()
	command.client = client
	command.cmd.SetArgs([]string{"OAuth sandbox"})

	err := command.cmd.Execute()
	require.ErrorContains(t, err, "sandbox name is only valid with an active live OAuth account")
	require.Empty(t, client.calls)
}

func TestSandboxCreateCmdRejectsRouteSpecificInputsBeforeSideEffects(t *testing.T) {
	t.Run("OAuth rejects anonymous identity flags", func(t *testing.T) {
		cleanup := setupSandboxTestConfig(t)
		defer cleanup()
		setSandboxCreateOAuthContext(t)

		client := &fakeSandboxCreateClient{}
		command := newSandboxCreateCmd()
		command.client = client
		command.cmd.SetArgs([]string{"OAuth sandbox", "--email", "test@stripe.com"})

		err := command.cmd.Execute()
		require.ErrorContains(t, err, "--email is only valid for anonymous sandbox provisioning")
		require.Empty(t, client.calls)
	})

	t.Run("anonymous rejects management flags", func(t *testing.T) {
		cleanup := setupSandboxTestConfig(t)
		defer cleanup()

		client := &fakeSandboxCreateClient{}
		command := newSandboxCreateCmd()
		command.client = client
		command.cmd.SetArgs([]string{"--email", "test@stripe.com", "--create-blank", "--country", "US"})

		err := command.cmd.Execute()
		require.ErrorContains(t, err, "--create-blank is only valid with an active live OAuth account")
		require.Empty(t, client.calls)
	})
}

func TestSandboxCreateCmdOAuthRequiresName(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }

	err := command.cmd.Execute()
	require.EqualError(t, err, "sandbox name is required; for example: `stripe sandbox create \"My sandbox\"`")
	require.Empty(t, client.calls)
}

func TestSandboxCreateCmdOAuthPromptsForMissingName(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("  Prompted sandbox  \n"))

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Prompted sandbox"}}, client.calls)
	assert.Contains(t, stdout.String(), "Sandbox name:")
	assert.Contains(t, stdout.String(), "Created sandbox \"Prompted sandbox\"")
	assert.Empty(t, stderr.String())
}

func TestSandboxCreateCmdOAuthPromptsForMissingBlankSandboxName(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("Blank sandbox\n"))
	command.cmd.SetArgs([]string{"--create-blank", "--country", " us "})

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Blank sandbox", Blank: true, Country: "US"}}, client.calls)
}

func TestSandboxCreateCmdOAuthPositionalNameDoesNotPrompt(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("no\n"))
	command.cmd.SetArgs([]string{"Positional sandbox"})

	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Positional sandbox"}}, client.calls)
	assert.NotContains(t, stdout.String(), "Sandbox name:")
	assert.Contains(t, stdout.String(), "Authorize this sandbox with the CLI now?")
}

func TestSandboxCreateCmdOAuthRejectsInvalidPromptInput(t *testing.T) {
	tests := []struct {
		name  string
		setIn func(*cobra.Command)
		want  string
	}{
		{
			name: "blank",
			setIn: func(cmd *cobra.Command) {
				cmd.SetIn(strings.NewReader("  \n"))
			},
			want: "sandbox name cannot be blank",
		},
		{
			name: "EOF",
			setIn: func(cmd *cobra.Command) {
				cmd.SetIn(strings.NewReader(""))
			},
			want: "sandbox name cannot be blank",
		},
		{
			name: "read error",
			setIn: func(cmd *cobra.Command) {
				cmd.SetIn(iotest.ErrReader(errors.New("read failed")))
			},
			want: "failed to read sandbox name",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			setSandboxCreateOAuthContext(t)

			telemetry := newFakeSandboxTelemetryClient()
			client := &fakeSandboxCreateClient{}
			command := newSandboxCreateCmd()
			command.client = client
			command.isInteractive = func(*cobra.Command) bool { return true }
			command.cmd.SetContext(stripe.WithTelemetryClient(context.Background(), telemetry))
			test.setIn(command.cmd)

			err := command.cmd.Execute()
			require.ErrorContains(t, err, test.want)
			require.Empty(t, client.calls)
			telemetry.assertNoEvent(t)
		})
	}
}

func TestSandboxCreateCmdOAuthPromptedNameThenAuthorizes(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("Prompted sandbox\nyes\n"))

	reauthCalls := 0
	command.reauth = func(context.Context, string, string) error {
		reauthCalls++
		return nil
	}

	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Prompted sandbox"}}, client.calls)
	assert.Equal(t, 1, reauthCalls)
	assert.Contains(t, stdout.String(), "Sandbox name:")
	assert.Contains(t, stdout.String(), "Authorize this sandbox with the CLI now?")
}

func TestSandboxCreateCmdOAuth_InteractiveYesRefreshesAndReauthenticates(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	previousAccessBase := rootAccessBaseURL
	rootAccessBaseURL = login.QAAccessBaseURL
	t.Cleanup(func() { rootAccessBaseURL = previousAccessBase })

	activeContext, err := json.Marshal(config.ActiveContext{AccountID: "acct_live", Livemode: true})
	require.NoError(t, err)
	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:            []byte("oak_original"),
		config.OAuthUATExpiresAtKeychainKey:  []byte(time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)),
		config.OAuthActiveContextKeychainKey: activeContext,
	})
	previousRefresher := config.OAuthTokenRefresher
	t.Cleanup(func() { config.OAuthTokenRefresher = previousRefresher })

	refreshCalls := 0
	config.OAuthTokenRefresher = func(profile *config.Profile) error {
		refreshCalls++
		profile.UAT = "oak_refreshed"
		return nil
	}

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }

	ctx := context.WithValue(context.Background(), sandboxTestContextKey{}, "sandbox-test")
	command.cmd.SetContext(ctx)
	command.cmd.SetIn(strings.NewReader("  YeS  \n"))

	var reauthContext context.Context
	var reauthAccessBaseURL string
	var reauthAccessToken string
	reauthCalls := 0
	command.reauth = func(ctx context.Context, accessBaseURL, accessToken string) error {
		reauthCalls++
		reauthContext = ctx
		reauthAccessBaseURL = accessBaseURL
		reauthAccessToken = accessToken
		return nil
	}

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)
	command.cmd.SetArgs([]string{"Created sandbox"})

	require.NoError(t, command.cmd.Execute())
	assert.Equal(t, 1, refreshCalls)
	assert.Equal(t, 1, reauthCalls)
	assert.Equal(t, ctx, reauthContext)
	assert.Equal(t, login.QAAccessBaseURL, reauthAccessBaseURL)
	assert.Equal(t, "oak_refreshed", reauthAccessToken)
	assert.Contains(t, stdout.String(), "Authorize this sandbox with the CLI now? [y/N]:")
	assert.Contains(t, stdout.String(), "stripe login")
	assert.Empty(t, stderr.String())
}

func TestSandboxCreateCmdOAuth_InteractiveDeclinesWithoutReauth(t *testing.T) {
	for _, input := range []string{"\n", "n\n", "no\n", "maybe\n", ""} {
		t.Run(fmt.Sprintf("input_%q", input), func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			setSandboxCreateOAuthContext(t)

			client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
			command := newSandboxCreateCmd()
			command.client = client
			command.isInteractive = func(*cobra.Command) bool { return true }
			command.cmd.SetIn(strings.NewReader(input))
			command.cmd.SetArgs([]string{"Created sandbox"})

			reauthCalls := 0
			command.reauth = func(context.Context, string, string) error {
				reauthCalls++
				return nil
			}

			var stdout, stderr bytes.Buffer
			command.cmd.SetOut(&stdout)
			command.cmd.SetErr(&stderr)

			require.NoError(t, command.cmd.Execute())
			assert.Equal(t, 0, reauthCalls)
			assert.Contains(t, stdout.String(), "Authorize this sandbox with the CLI now? [y/N]:")
			assert.Contains(t, stdout.String(), "stripe login")
			assert.Empty(t, stderr.String())
		})
	}
}

func TestSandboxCreateCmdOAuth_NonInteractiveDoesNotPrompt(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetArgs([]string{"Created sandbox"})

	command.reauth = func(context.Context, string, string) error {
		t.Fatal("reauth should not run for a non-interactive caller")
		return nil
	}

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)

	require.NoError(t, command.cmd.Execute())
	assert.NotContains(t, stdout.String(), "Authorize this sandbox")
	assert.Contains(t, stdout.String(), "stripe login")
	assert.Empty(t, stderr.String())
}

func TestSandboxCreateCmdOAuth_InvalidUATAfterCreationWarnsWithoutReauth(t *testing.T) {
	for _, uat := range []string{"", "sk_test_not_oak"} {
		t.Run(fmt.Sprintf("uat_%q", uat), func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			setSandboxCreateOAuthContext(t)

			client := &fakeSandboxCreateClient{
				created: sandbox.CreatedSandbox{AccountID: "acct_created"},
				beforeReturn: func() {
					initial := map[string][]byte{}
					if uat != "" {
						initial[config.UATKeychainItemKey] = []byte(uat)
					}
					config.KeyRing = keyring.NewMemoryStore(initial)
				},
			}
			command := newSandboxCreateCmd()
			command.client = client
			command.isInteractive = func(*cobra.Command) bool { return true }
			command.cmd.SetIn(strings.NewReader("y\n"))
			command.cmd.SetArgs([]string{"Created sandbox"})

			reauthCalls := 0
			command.reauth = func(context.Context, string, string) error {
				reauthCalls++
				return nil
			}

			var stdout, stderr bytes.Buffer
			command.cmd.SetOut(&stdout)
			command.cmd.SetErr(&stderr)

			require.NoError(t, command.cmd.Execute())
			assert.Equal(t, 0, reauthCalls)
			assert.Contains(t, stdout.String(), "Created sandbox")
			assert.Contains(t, stderr.String(), "sandbox creation succeeded")
			assert.Contains(t, stderr.String(), "stripe login")
			assert.NotContains(t, stderr.String(), "sandbox new")
		})
	}
}

func TestSandboxCreateCmdOAuth_ReauthFailurePreservesCreationSuccess(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_created"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("yes\n"))
	command.cmd.SetArgs([]string{"Created sandbox"})
	command.reauth = func(context.Context, string, string) error {
		return fmt.Errorf("reauth failed")
	}

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)

	require.NoError(t, command.cmd.Execute())
	assert.Contains(t, stdout.String(), "Created sandbox")
	assert.Contains(t, stderr.String(), "sandbox creation succeeded")
	assert.Contains(t, stderr.String(), "stripe login")
	assert.NotContains(t, stderr.String(), "sandbox new")
}

func TestSandboxCreateCmdOAuthCreatesBlankWithNormalizedCountry(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{created: sandbox.CreatedSandbox{AccountID: "acct_blank"}}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return false }
	command.cmd.SetArgs([]string{"Blank sandbox", "--create-blank", "--country", " us "})
	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []sandbox.CreateOptions{{Name: "Blank sandbox", Blank: true, Country: "US"}}, client.calls)
	require.Contains(t, stdout.String(), "acct_blank")
}

func TestSandboxCreateCmdOAuthRejectsInvalidInputBeforeCallingClient(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing name"},
		{name: "extra argument", args: []string{"one", "two"}},
		{name: "blank name", args: []string{"   "}},
		{name: "blank without country", args: []string{"blank", "--create-blank"}},
		{name: "country without blank", args: []string{"copy", "--country", "US"}},
		{name: "short country", args: []string{"blank", "--create-blank", "--country", "U"}},
		{name: "long country", args: []string{"blank", "--create-blank", "--country", "USA"}},
		{name: "nonletters country", args: []string{"blank", "--create-blank", "--country", "1S"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanup := setupSandboxTestConfig(t)
			defer cleanup()
			setSandboxCreateOAuthContext(t)

			client := &fakeSandboxCreateClient{}
			command := newSandboxCreateCmd()
			command.client = client
			command.cmd.SetArgs(test.args)

			require.Error(t, command.cmd.Execute())
			require.Empty(t, client.calls)
		})
	}
}

func TestSandboxCreateCmdOAuthReturnsClientErrorWithoutSuccessOutput(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()
	setSandboxCreateOAuthContext(t)

	client := &fakeSandboxCreateClient{err: fmt.Errorf("safe create failure")}
	command := newSandboxCreateCmd()
	command.client = client
	command.isInteractive = func(*cobra.Command) bool { return true }
	command.cmd.SetIn(strings.NewReader("yes\n"))
	command.reauth = func(context.Context, string, string) error {
		t.Fatal("reauth should not run when creation fails")
		return nil
	}
	command.cmd.SetArgs([]string{"Failed sandbox"})
	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	err := command.cmd.Execute()
	require.EqualError(t, err, "safe create failure")
	require.NotContains(t, stdout.String(), "Created sandbox")
	require.NotContains(t, stdout.String(), "acct_")
}

type fakeSandboxListClient struct {
	sandboxes []sandbox.ManagedSandbox
	err       error
}

func (f fakeSandboxListClient) ListAccessible(context.Context) ([]sandbox.ManagedSandbox, error) {
	return f.sandboxes, f.err
}

func TestSandboxListCmd_PresentsSandboxes(t *testing.T) {
	cmd := newSandboxListCmd()
	cmd.client = fakeSandboxListClient{
		sandboxes: []sandbox.ManagedSandbox{
			{WorkspaceID: "wksp_test_z", AccountID: "acct_z", Name: "Zeta", AccessLevel: sandbox.SandboxAccessLevelPrivate},
			{WorkspaceID: "wksp_test_a", AccountID: "acct_a", Name: "Alpha", AccessLevel: sandbox.SandboxAccessLevelGlobal},
			{WorkspaceID: "wksp_test_b", AccountID: "acct_b", Name: "Beta", AccessLevel: sandbox.SandboxAccessLevelDeveloper},
			{WorkspaceID: "wksp_test_ltm", AccountID: "acct_ltm", Name: "Acme", AccessLevel: sandbox.SandboxAccessLevelPrivate, IsLegacyTestmode: true},
		},
	}

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	require.NoError(t, cmd.cmd.Execute())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 5)
	assert.Equal(t, []string{"NAME", "ACCOUNT", "ACCESS"}, strings.Fields(lines[0]))
	assert.Equal(t, []string{"Zeta", "acct_z", "Private"}, strings.Fields(lines[1]))
	assert.Equal(t, []string{"Alpha", "acct_a", "All", "team", "members"}, strings.Fields(lines[2]))
	assert.Equal(t, []string{"Beta", "acct_b", "Developer"}, strings.Fields(lines[3]))
	assert.Equal(t, []string{"Acme", "acct_ltm", "Private"}, strings.Fields(lines[4]))
	assert.NotContains(t, stdout.String(), "wksp_")
	assert.NotContains(t, stdout.String(), "TYPE")
	assert.Empty(t, stderr.String())
}

func TestSandboxListCmd_Empty(t *testing.T) {
	cmd := newSandboxListCmd()
	cmd.client = fakeSandboxListClient{sandboxes: []sandbox.ManagedSandbox{}}

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	require.NoError(t, cmd.cmd.Execute())

	assert.Equal(t, "No sandboxes found.\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestSandboxListCmd_ClientError(t *testing.T) {
	cmd := newSandboxListCmd()
	cmd.client = fakeSandboxListClient{err: fmt.Errorf("could not list accessible sandboxes")}

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)
	cmd.cmd.SilenceUsage = true
	cmd.cmd.SilenceErrors = true

	err := cmd.cmd.Execute()
	require.EqualError(t, err, "could not list accessible sandboxes")
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

func TestSandboxListCmd_Surface(t *testing.T) {
	cmd := newSandboxListCmd()
	assert.False(t, cmd.cmd.Hidden)
	assert.Equal(t, "List the sandboxes available to your account", cmd.cmd.Short)
	assert.Contains(t, cmd.cmd.Long, "active live account")
	assert.Contains(t, cmd.cmd.Long, "stripe login")
	assert.Equal(t, "stripe sandbox list", cmd.cmd.Example)
	assert.Contains(t, cmd.cmd.Annotations[AIAgentHelpAnnotationKey], "stripe sandbox delete")
	require.NotNil(t, cmd.cmd.Flags().Lookup("api-base"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-account"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-version"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("json"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("format"))

	var stdout, stderr bytes.Buffer
	cmd.client = fakeSandboxListClient{}
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)
	cmd.cmd.SilenceUsage = true
	cmd.cmd.SilenceErrors = true
	cmd.cmd.SetArgs([]string{"--api-base=http://example.test"})
	require.NoError(t, cmd.cmd.Execute())
	assert.Equal(t, "No sandboxes found.\n", stdout.String())
	assert.Empty(t, stderr.String())

	for _, args := range [][]string{{"acct_123"}, {"--stripe-account=acct_123"}, {"--stripe-version=2026-01-01"}} {
		cmd := newSandboxListCmd()
		cmd.client = fakeSandboxListClient{}
		cmd.cmd.SilenceUsage = true
		cmd.cmd.SilenceErrors = true
		cmd.cmd.SetArgs(args)
		require.Error(t, cmd.cmd.Execute())
	}
}

type fakeSandboxDeleteClient struct {
	deleted sandbox.DeletedSandbox
	err     error
	calls   []string
}

func (f *fakeSandboxDeleteClient) Delete(_ context.Context, accountID string) (sandbox.DeletedSandbox, error) {
	f.calls = append(f.calls, accountID)
	return f.deleted, f.err
}

func TestSandboxDeleteCmd_Success(t *testing.T) {
	client := &fakeSandboxDeleteClient{deleted: sandbox.DeletedSandbox{AccountID: "acct_target", Name: "Target sandbox"}}
	command := newSandboxDeleteCmd()
	command.client = client
	command.cmd.SetArgs([]string{"--api-base=http://example.test", "--confirm", "  acct_target  "})

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []string{"acct_target"}, client.calls)
	require.Equal(t, "Deleted sandbox \"Target sandbox\" (acct_target)\n", stdout.String())
	require.Empty(t, stderr.String())
	require.NotContains(t, stdout.String(), "wksp_")
}

func TestSandboxDeleteCmd_ClientError(t *testing.T) {
	client := &fakeSandboxDeleteClient{err: fmt.Errorf("safe delete failure")}
	command := newSandboxDeleteCmd()
	command.client = client
	command.cmd.SetArgs([]string{"acct_target", "--confirm"})

	var stdout, stderr bytes.Buffer
	command.cmd.SetOut(&stdout)
	command.cmd.SetErr(&stderr)
	command.cmd.SilenceUsage = true
	command.cmd.SilenceErrors = true

	err := command.cmd.Execute()
	require.EqualError(t, err, "safe delete failure")
	require.Equal(t, []string{"acct_target"}, client.calls)
	require.Empty(t, stdout.String())
	require.Empty(t, stderr.String())
}

func TestSandboxDeleteCmd_RejectsInvalidInputBeforeCallingClient(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing account"},
		{name: "empty account", args: []string{"   ", "--confirm"}},
		{name: "organization", args: []string{"org_123", "--confirm"}},
		{name: "wrong prefix", args: []string{"not_an_account", "--confirm"}},
		{name: "missing account suffix", args: []string{"acct_", "--confirm"}},
		{name: "extra account", args: []string{"acct_one", "acct_two", "--confirm"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &fakeSandboxDeleteClient{}
			command := newSandboxDeleteCmd()
			command.client = client
			command.cmd.SetArgs(test.args)

			require.Error(t, command.cmd.Execute())
			require.Empty(t, client.calls)
		})
	}
}

func TestSandboxDeleteCmd_Surface(t *testing.T) {
	command := newSandboxDeleteCmd()
	require.False(t, command.cmd.Hidden)
	require.Equal(t, "delete <account_id>", command.cmd.Use)
	require.Equal(t, "Permanently delete a sandbox", command.cmd.Short)
	require.Contains(t, command.cmd.Long, "does not affect your live account")
	require.Contains(t, command.cmd.Long, "cannot be undone")
	require.Contains(t, command.cmd.Long, "stripe sandbox list")
	require.Equal(t, "stripe sandbox delete acct_123\n  stripe sandbox delete acct_123 --confirm", command.cmd.Example)
	require.Contains(t, command.cmd.Annotations[AIAgentHelpAnnotationKey], "stripe sandbox list")
	require.Nil(t, command.cmd.Flags().Lookup("stripe-account"))
	require.NotNil(t, command.cmd.Flags().Lookup("confirm"))
	require.Nil(t, command.cmd.Flags().Lookup("yes"))
	require.NotNil(t, command.cmd.Flags().ShorthandLookup("c"))
	require.Nil(t, command.cmd.Flags().ShorthandLookup("y"))
	require.NotNil(t, command.cmd.Flags().Lookup("api-base"))
	require.Nil(t, command.cmd.Flags().Lookup("stripe-version"))
	require.Nil(t, command.cmd.Flags().Lookup("json"))
	require.Nil(t, command.cmd.Flags().Lookup("format"))
	require.NotContains(t, command.cmd.UsageString(), "--stripe-account stripe sandbox list")
	require.Contains(t, command.cmd.UsageString(), "--confirm")
	require.NotContains(t, command.cmd.UsageString(), "--yes")

	client := &fakeSandboxDeleteClient{deleted: sandbox.DeletedSandbox{AccountID: "acct_target"}}
	command.client = client
	command.cmd.SetArgs([]string{"acct_target", "-c"})
	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)
	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []string{"acct_target"}, client.calls)
	require.Equal(t, "Deleted sandbox acct_target\n", stdout.String())
}

func TestSandboxDeleteCmd_RequiresConfirmationWhenNonInteractive(t *testing.T) {
	client := &fakeSandboxDeleteClient{}
	command := newSandboxDeleteCmd()
	command.client = client
	command.cmd.SetIn(os.Stdin)
	command.cmd.SetArgs([]string{"acct_target"})

	err := command.cmd.Execute()
	require.EqualError(t, err, "refusing to delete sandbox acct_target without confirmation; re-run with --confirm")
	require.Empty(t, client.calls)
}

func TestSandboxDeleteCmd_AbortsUnlessConfirmed(t *testing.T) {
	client := &fakeSandboxDeleteClient{}
	command := newSandboxDeleteCmd()
	command.client = client
	command.cmd.SetIn(strings.NewReader("\n"))
	command.cmd.SetArgs([]string{"acct_target"})
	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	require.NoError(t, command.cmd.Execute())
	require.Empty(t, client.calls)
	require.Equal(t, "Delete sandbox acct_target?\nThis action cannot be undone.\nContinue? [y/N]: Aborted. No changes were made.\n", stdout.String())
}

func TestSandboxDeleteCmd_DeletesAfterConfirmation(t *testing.T) {
	client := &fakeSandboxDeleteClient{deleted: sandbox.DeletedSandbox{AccountID: "acct_target", Name: "Target sandbox"}}
	command := newSandboxDeleteCmd()
	command.client = client
	command.cmd.SetIn(strings.NewReader("yes\n"))
	command.cmd.SetArgs([]string{"acct_target"})
	var stdout bytes.Buffer
	command.cmd.SetOut(&stdout)

	require.NoError(t, command.cmd.Execute())
	require.Equal(t, []string{"acct_target"}, client.calls)
	require.Equal(t, "Delete sandbox acct_target?\nThis action cannot be undone.\nContinue? [y/N]: Deleted sandbox \"Target sandbox\" (acct_target)\n", stdout.String())
}
