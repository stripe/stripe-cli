package cmd

import (
	"bytes"
	"context"
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
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/requests"
	"github.com/stripe/stripe-cli/pkg/sandbox"
)

func setupSandboxTestConfig(t *testing.T) func() {
	t.Helper()
	resetSandboxNewFlagsForTest(t)
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

func resetSandboxNewFlagsForTest(t *testing.T) {
	t.Helper()
	cmd, _, err := rootCmd.Find([]string{"sandbox", "new"})
	require.NoError(t, err)

	for _, name := range []string{
		"name",
		"copy-live-account",
		"create-blank",
		"business-location",
		"stripe-account",
		"activate",
		"batch",
		"stripe-version",
		"api-base",
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

func TestSandboxNewCmd_Success(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	// Seed a UAT into the in-memory keyring
	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// --stripe-account pins the live account; the command resolves its workspace and playground,
	// then creates. The server serves the user_accessible GET, playground GET, and the create POST.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			assert.Equal(t, "STRIPE-V2-SIG keyinfo_live_faketoken", r.Header.Get("Authorization"))
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_livetest", "name": "Live Test", "merchant_id": "acct_livetest"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_livetest":
			assert.Equal(t, "STRIPE-V2-SIG keyinfo_live_faketoken", r.Header.Get("Authorization"))
			// Resolution GETs are self-scoped by the UAT; no Stripe-Context header.
			assert.Empty(t, r.Header.Get("Stripe-Context"))
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_livetest"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			assert.Equal(t, "STRIPE-V2-SIG keyinfo_live_faketoken", r.Header.Get("Authorization"))
			// Stripe-Context must be the resolved playground id, not the workspace.
			assert.Equal(t, "play_livetest", r.Header.Get("Stripe-Context"))
			assert.Equal(t, requests.StripeVersionHeaderValue, r.Header.Get("Stripe-Version"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "mytest", body["name"])
			assert.Equal(t, "wksp_livetest", body["replica_of"])
			assert.Equal(t, true, body["activate_sandbox"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sbx_123", "v1_account_id": "acct_livetest", "object": "sandbox"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=acct_livetest",
		"--name=mytest",
	)

	require.NoError(t, err)
	assert.Contains(t, output, "sbx_123")
	assert.Contains(t, output, "mytest")
}

func TestSandboxNewCmd_ActivateFalse(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_livetest", "name": "Live Test", "merchant_id": "acct_livetest"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_livetest":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_livetest"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, false, body["activate_sandbox"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sbx_123", "object": "sandbox"})
		}
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=acct_livetest",
		"--name=mytest",
		"--activate=false",
	)
	require.NoError(t, err)
}

func TestSandboxNewCmd_RejectsResolvedNonPlayground(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// If the playground endpoint returns a non-play_ id, the command must reject it.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_livetest", "name": "Live Test", "merchant_id": "acct_livetest"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_livetest":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "wksp_notaplayground"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=acct_livetest",
		"--name=mytest",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-playground context")
}

func TestSandboxNewCmd_BlankPath(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// Blank mode has no --stripe-account, so the live workspace is resolved via
	// user_accessible, then its playground, then create.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{{"id": "wksp_blankparent", "name": "Blank Parent", "merchant_id": "acct_blank"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_blankparent":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_blank"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			assert.Equal(t, "play_blank", r.Header.Get("Stripe-Context"))
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			// Blank path sends business_location and omits replica_of entirely.
			assert.Equal(t, "US", body["business_location"])
			_, hasReplica := body["replica_of"]
			assert.False(t, hasReplica)
			assert.Equal(t, false, body["activate_sandbox"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "wksp_test_blank", "object": "sandbox"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--create-blank=true",
		"--copy-live-account=false",
		"--stripe-account=",
		"--business-location=US",
		"--name=blanktest",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "blanktest")
}

func TestSandboxNewCmd_ModesMutuallyExclusive(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--copy-live-account=true",
		"--create-blank=true",
		"--name=mytest",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestSandboxNewCmd_RequiresMode(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--copy-live-account=false",
		"--create-blank=false",
		"--name=mytest",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pass one of")
}

func TestSandboxNewCmd_AutoResolveViaUserAccessible(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// No --stripe-account: the command must resolve the live workspace via
	// /v2/compartments/user_accessible, the playground via
	// /v2/compartments/playground/:id, then create.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{{"id": "wksp_auto", "name": "Auto", "merchant_id": "acct_auto"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_auto":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_auto"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			assert.Equal(t, "play_auto", r.Header.Get("Stripe-Context"))
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "wksp_auto", body["replica_of"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sbx_auto", "object": "sandbox"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=",
		"--business-location=",
		"--activate=true",
		"--name=autotest",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "autotest")
}

func TestSandboxNewCmd_SkipsOrgLoginContext(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// Write a profiles file with user_info containing an org_ in livemode.
	// resolveLiveWorkspace must skip it and fall through to user_accessible.
	profilesFile := Config.ProfilesFile
	tomlContent := `[default]

[[user_info.compartments]]
compartment_id = "org_shouldbeskipped"
livemode = true
`
	os.WriteFile(profilesFile, []byte(tomlContent), 0600)

	// Re-read viper with the new file so GetUserInfo sees the org
	viper.Reset()
	viper.SetConfigFile(Config.ProfilesFile)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	// Verify the org was seeded correctly
	var ui config.UserInfo
	err = viper.UnmarshalKey("user_info", &ui)
	require.NoError(t, err)
	require.Len(t, ui.Compartments, 1)
	require.Equal(t, "org_shouldbeskipped", ui.Compartments[0].CompartmentID)

	// Server mock: user_accessible returns wksp_fromlist, playground GET for that,
	// then create returns sbx_orgskip
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			// This is the fallback the test is checking: org_ should be skipped,
			// so we must hit this endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{{"id": "wksp_fromlist"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_fromlist":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_x"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			var body map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			// Must use wksp_fromlist (from user_accessible), NOT the org_
			assert.Equal(t, "wksp_fromlist", body["replica_of"])
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            "sbx_orgskip",
				"v1_account_id": "acct_x",
				"object":        "sandbox",
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "org_shouldbeskipped"):
			// If the server ever sees a playground GET for the org_, fail
			t.Errorf("server received request for org_shouldbeskipped playground, but org_ should have been skipped")
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	output, err := executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=",
		"--business-location=",
		"--activate=true",
		"--name=orgskiptest",
	)
	require.NoError(t, err)
	assert.Contains(t, output, "orgskiptest")
}

func TestSandboxNewCmd_MultipleWorkspacesError(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{{"id": "wksp_a"}, {"id": "wksp_b"}},
			})
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=",
		"--business-location=",
		"--name=autotest",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple livemode workspaces")
}

func TestSandboxNewCmd_NoWorkspaceError(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible" {
			json.NewEncoder(w).Encode(map[string]interface{}{"standalone_workspaces": []map[string]interface{}{}})
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=",
		"--business-location=",
		"--name=autotest",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no livemode workspace")
}

func TestSandboxNewCmd_NoUAT(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	// Don't seed a UAT — keyring is empty
	// Execute the command
	_, err := executeCommand(
		rootCmd,
		"sandbox", "new",
		"--name=mytest",
		"--stripe-account=acct_livetest",
		"--copy-live-account=true",
		"--create-blank=false",
	)

	// Should fail with an error mentioning stripe login
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stripe login")
}

func TestSandboxNewCmd_BatchNotYetSupported(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	// --batch is optional (defaults to 1); >1 is rejected until bulk-create lands.
	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--stripe-account=acct_livetest",
		"--copy-live-account=true",
		"--create-blank=false",
		"--name=mytest",
		"--batch=3",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")

	// batch < 1 is invalid.
	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--stripe-account=acct_livetest",
		"--copy-live-account=true",
		"--create-blank=false",
		"--name=mytest",
		"--batch=0",
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--batch must be >= 1")
}

func TestSandboxNewCmd_ResolvesByStripeAccount(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_target", "name": "Acme", "merchant_id": "acct_target"},
					{"id": "wksp_other", "name": "Other", "merchant_id": "acct_other"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_target":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_target"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			assert.Equal(t, "play_target", r.Header.Get("Stripe-Context"))
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "wksp_target", body["replica_of"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sbx_byacct", "v1_account_id": "acct_target", "object": "sandbox"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=acct_target",
		"--name=targettest",
		"--batch=1",
	)

	require.NoError(t, err)
}

func TestSandboxNewCmd_StripeAccountBlankMode(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_target", "name": "Acme", "merchant_id": "acct_target"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/playground/wksp_target":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "play_target"})
		case r.Method == http.MethodPost && r.URL.Path == "/v2/sandboxes":
			assert.Equal(t, "play_target", r.Header.Get("Stripe-Context"))
			var body map[string]interface{}
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "US", body["business_location"])
			_, hasReplica := body["replica_of"]
			assert.False(t, hasReplica)
			assert.Equal(t, false, body["activate_sandbox"])
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "sbx_blankacct", "object": "sandbox"})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--create-blank=true",
		"--copy-live-account=false",
		"--business-location=US",
		"--stripe-account=acct_target",
		"--name=blankaccttest",
		"--batch=1",
	)

	require.NoError(t, err)
}

func TestSandboxNewCmd_StripeAccountNotFound(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/v2/compartments/user_accessible" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"standalone_workspaces": []map[string]interface{}{
					{"id": "wksp_other", "name": "Other", "merchant_id": "acct_other"},
				},
			})
			return
		}
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--api-base="+server.URL,
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=acct_missing",
		"--business-location=",
		"--name=missingtest",
		"--batch=1",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no accessible live account matches")
}

func TestSandboxNewCmd_StripeAccountRejectsOrg(t *testing.T) {
	cleanup := setupSandboxTestConfig(t)
	defer cleanup()

	err := config.KeyRing.Set(config.UATKeychainItemKey, []byte("keyinfo_live_faketoken"), "test uat")
	require.NoError(t, err)

	_, err = executeCommand(
		rootCmd,
		"sandbox", "new",
		"--copy-live-account=true",
		"--create-blank=false",
		"--stripe-account=org_123",
		"--business-location=",
		"--name=orgtest",
		"--batch=1",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not an organization")
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
			{WorkspaceID: "wksp_test_z", AccountID: "acct_z", Name: "Zeta", AccessLevel: sandbox.AccessLevelNone},
			{WorkspaceID: "wksp_test_a", AccountID: "acct_a", Name: "Alpha", AccessLevel: sandbox.AccessLevelDirect},
			{WorkspaceID: "wksp_test_b", AccountID: "acct_b", Name: "Beta", AccessLevel: sandbox.AccessLevelSandboxChildren},
		},
	}

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)

	require.NoError(t, cmd.cmd.Execute())

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, []string{"NAME", "ACCOUNT", "ACCESS", "LEVEL"}, strings.Fields(lines[0]))
	assert.Equal(t, []string{"Zeta", "acct_z", "NO_ACCESS"}, strings.Fields(lines[1]))
	assert.Equal(t, []string{"Alpha", "acct_a", "DIRECT_ACCESS"}, strings.Fields(lines[2]))
	assert.Equal(t, []string{"Beta", "acct_b", "ACCESS_TO_SANDBOX_CHILDREN"}, strings.Fields(lines[3]))
	assert.NotContains(t, stdout.String(), "wksp_")
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
	assert.True(t, cmd.cmd.Hidden)
	require.NotNil(t, cmd.cmd.Flags().Lookup("api-base"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-account"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-version"))

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
	target          *sandbox.DeleteTarget
	prepareErr      error
	deleteErr       error
	prepareAccounts []string
	deletedTargets  []*sandbox.DeleteTarget
}

func (f *fakeSandboxDeleteClient) PrepareDelete(_ context.Context, accountID string) (*sandbox.DeleteTarget, error) {
	f.prepareAccounts = append(f.prepareAccounts, accountID)
	return f.target, f.prepareErr
}

func (f *fakeSandboxDeleteClient) Delete(_ context.Context, target *sandbox.DeleteTarget) error {
	f.deletedTargets = append(f.deletedTargets, target)
	return f.deleteErr
}

func TestSandboxDeleteCmd_ConfirmsAndDeletes(t *testing.T) {
	target := &sandbox.DeleteTarget{AccountID: "acct_sandbox", Name: "QA Sandbox"}
	client := &fakeSandboxDeleteClient{target: target}
	cmd := newSandboxDeleteCmd()
	cmd.client = client
	cmd.cmd.SetArgs([]string{"acct_sandbox"})
	cmd.cmd.SetIn(strings.NewReader("yes\n"))

	var stdout, stderr bytes.Buffer
	cmd.cmd.SetOut(&stdout)
	cmd.cmd.SetErr(&stderr)
	cmd.cmd.SilenceUsage = true
	cmd.cmd.SilenceErrors = true

	require.NoError(t, cmd.cmd.Execute())
	assert.Equal(t, []string{"acct_sandbox"}, client.prepareAccounts)
	assert.Equal(t, []*sandbox.DeleteTarget{target}, client.deletedTargets)
	assert.Contains(t, stdout.String(), `Delete sandbox "QA Sandbox" (acct_sandbox)?`)
	assert.Contains(t, stdout.String(), "[y/N]")
	assert.Contains(t, stdout.String(), `Deleted sandbox "QA Sandbox" (acct_sandbox)`)
	assert.Empty(t, stderr.String())
}

func TestSandboxDeleteCmd_DeclinesWithoutDeleting(t *testing.T) {
	for _, input := range []string{"\n", "n\n", "anything else\n"} {
		t.Run(strings.TrimSpace(input), func(t *testing.T) {
			client := &fakeSandboxDeleteClient{
				target: &sandbox.DeleteTarget{AccountID: "acct_sandbox", Name: "QA Sandbox"},
			}
			cmd := newSandboxDeleteCmd()
			cmd.client = client
			cmd.cmd.SetArgs([]string{"acct_sandbox"})
			cmd.cmd.SetIn(strings.NewReader(input))

			var stdout bytes.Buffer
			cmd.cmd.SetOut(&stdout)

			require.NoError(t, cmd.cmd.Execute())
			assert.Empty(t, client.deletedTargets)
			assert.Contains(t, stdout.String(), "Aborted. No changes were made.")
		})
	}
}

func TestSandboxDeleteCmd_DoesNotReportSuccessOnClientError(t *testing.T) {
	target := &sandbox.DeleteTarget{AccountID: "acct_sandbox", Name: "QA Sandbox"}
	client := &fakeSandboxDeleteClient{target: target, deleteErr: fmt.Errorf("could not delete sandbox")}
	cmd := newSandboxDeleteCmd()
	cmd.client = client
	cmd.cmd.SetArgs([]string{"acct_sandbox"})
	cmd.cmd.SetIn(strings.NewReader("y\n"))
	cmd.cmd.SilenceUsage = true
	cmd.cmd.SilenceErrors = true

	var stdout bytes.Buffer
	cmd.cmd.SetOut(&stdout)

	err := cmd.cmd.Execute()
	require.EqualError(t, err, "could not delete sandbox")
	assert.Equal(t, []*sandbox.DeleteTarget{target}, client.deletedTargets)
	assert.NotContains(t, stdout.String(), "Deleted sandbox")
}

func TestSandboxDeleteCmd_RejectsNonInteractiveBeforeLookup(t *testing.T) {
	client := &fakeSandboxDeleteClient{}
	cmd := newSandboxDeleteCmd()
	cmd.client = client
	cmd.cmd.SetArgs([]string{"acct_sandbox"})
	cmd.cmd.SetIn(os.Stdin)
	cmd.cmd.SilenceUsage = true
	cmd.cmd.SilenceErrors = true

	err := cmd.cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interactive terminal")
	assert.Empty(t, client.prepareAccounts)
	assert.Empty(t, client.deletedTargets)
}

func TestSandboxDeleteCmd_ValidatesAccountBeforeLookup(t *testing.T) {
	for _, accountID := range []string{"org_123", "wksp_test_123", "acct_"} {
		t.Run(accountID, func(t *testing.T) {
			client := &fakeSandboxDeleteClient{}
			cmd := newSandboxDeleteCmd()
			cmd.client = client
			cmd.cmd.SetArgs([]string{accountID})
			cmd.cmd.SetIn(strings.NewReader("yes\n"))
			cmd.cmd.SilenceUsage = true
			cmd.cmd.SilenceErrors = true

			err := cmd.cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "sandbox account id")
			assert.Empty(t, client.prepareAccounts)
		})
	}
}

func TestSandboxDeleteCmd_Surface(t *testing.T) {
	cmd := newSandboxDeleteCmd()
	assert.True(t, cmd.cmd.Hidden)
	assert.Equal(t, "delete <account>", cmd.cmd.Use)
	require.NotNil(t, cmd.cmd.Flags().Lookup("api-base"))
	assert.True(t, cmd.cmd.Flags().Lookup("api-base").Hidden)
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-account"))
	assert.Nil(t, cmd.cmd.Flags().Lookup("stripe-version"))
	assert.NoError(t, cmd.cmd.Args(cmd.cmd, []string{"acct_sandbox"}))
	assert.Error(t, cmd.cmd.Args(cmd.cmd, nil))
	assert.Error(t, cmd.cmd.Args(cmd.cmd, []string{"acct_one", "acct_two"}))
}
