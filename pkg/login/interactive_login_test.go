package login

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99designs/keyring"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/acct"
)

const testAccountName = "test-account-name"

func setupInteractiveLoginConfig(t *testing.T) (*config.Config, func()) {
	t.Helper()
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	c := &config.Config{
		Color:    "auto",
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "test",
		},
		ProfilesFile: profilesFile,
	}
	c.InitConfig()
	config.KeyRing = keyring.NewArrayKeyring([]keyring.Item{})
	cleanup := func() {
		os.Remove(profilesFile)
	}
	return c, cleanup
}

func accountHandler(displayName string, accountID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := &acct.Account{ID: accountID}
		account.Settings.Dashboard.DisplayName = displayName
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(account)
	}
}

func TestInteractiveLoginTestModeKey(t *testing.T) {
	ts := httptest.NewServer(accountHandler(testAccountName, "acct_123"))
	defer ts.Close()

	cfg, cleanup := setupInteractiveLoginConfig(t)
	defer cleanup()

	input := strings.NewReader("sk_test_foobar\n")
	err := interactiveLoginWithParams(context.Background(), cfg, input, ts.URL)
	require.NoError(t, err)

	require.Equal(t, "sk_test_foobar", cfg.Profile.TestModeAPIKey)
	require.Empty(t, cfg.Profile.LiveModeAPIKey)
}

func TestInteractiveLoginLiveModeKey(t *testing.T) {
	ts := httptest.NewServer(accountHandler(testAccountName, "acct_123"))
	defer ts.Close()

	cfg, cleanup := setupInteractiveLoginConfig(t)
	defer cleanup()

	input := strings.NewReader("sk_live_foobar\n")
	err := interactiveLoginWithParams(context.Background(), cfg, input, ts.URL)
	require.NoError(t, err)

	require.Equal(t, "sk_live_foobar", cfg.Profile.LiveModeAPIKey)
	require.Empty(t, cfg.Profile.TestModeAPIKey)
}

func TestInteractiveLoginAccountIDAndDisplayName(t *testing.T) {
	ts := httptest.NewServer(accountHandler(testAccountName, "acct_123"))
	defer ts.Close()

	cfg, cleanup := setupInteractiveLoginConfig(t)
	defer cleanup()

	input := strings.NewReader("sk_test_foobar\n")
	err := interactiveLoginWithParams(context.Background(), cfg, input, ts.URL)
	require.NoError(t, err)

	require.Equal(t, "acct_123", cfg.Profile.AccountID)
	require.Equal(t, testAccountName, cfg.Profile.DisplayName)
}

func TestAPIKeyInput(t *testing.T) {
	expectedKey := "sk_test_foo1234"

	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := getConfigureAPIKey(keyInput)

	require.Equal(t, expectedKey, actualKey)
	require.NoError(t, err)
}

func TestAPIKeyInputEmpty(t *testing.T) {
	expectedKey := ""
	expectedErrorString := "API key is required, please provide your API key"

	keyInput := strings.NewReader(expectedKey + "\n")
	actualKey, err := getConfigureAPIKey(keyInput)

	require.Equal(t, expectedKey, actualKey)
	require.NotNil(t, err)
	require.EqualError(t, err, expectedErrorString)
}

func TestDeviceNameInput(t *testing.T) {
	expectedDeviceName := "Bender's Laptop"
	deviceNameInput := strings.NewReader(expectedDeviceName)

	actualDeviceName := getConfigureDeviceName(deviceNameInput)

	require.Equal(t, expectedDeviceName, actualDeviceName)
}

func TestDeviceNameAutoDetect(t *testing.T) {
	hostName, _ := os.Hostname()
	deviceNameInput := strings.NewReader("")

	actualDeviceName := getConfigureDeviceName(deviceNameInput)

	require.Equal(t, hostName, actualDeviceName)
}
