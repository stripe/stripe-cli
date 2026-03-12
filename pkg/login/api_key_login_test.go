package login

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login/acct"
)

func TestLoginWithAPIKeyDoesNotUseBrowserFlow(t *testing.T) {
	viper.Reset()
	defer viper.Reset()

	apiKey := "sk_test_1234567890"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "/v1/account", r.URL.Path)

		account := &acct.Account{ID: "acct_123"}
		account.Settings.Dashboard.DisplayName = "test-display"

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(account))
	}))
	defer ts.Close()

	profilesFile := filepath.Join(t.TempDir(), "stripe", "config.toml")
	viper.SetConfigFile(profilesFile)

	cfg := &config.Config{
		Color:    "auto",
		LogLevel: "info",
		Profile: config.Profile{
			DeviceName:  "st-testing",
			ProfileName: "default",
		},
		ProfilesFile: profilesFile,
	}
	cfg.InitConfig()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()

	os.Stdout = w
	err = LoginWithAPIKey(context.Background(), ts.URL, cfg, apiKey)
	_ = w.Close()
	os.Stdout = oldStdout
	require.NoError(t, err)

	outBytes, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	output := string(outBytes)

	require.NotContains(t, strings.ToLower(output), "pairing code")
	require.NotContains(t, output, "Press Enter")
	require.Contains(t, output, "Done! The Stripe CLI is configured")

	configBytes, fileErr := os.ReadFile(profilesFile)
	require.NoError(t, fileErr)
	require.Contains(t, string(configBytes), "test_mode_api_key")
	require.Contains(t, string(configBytes), apiKey)
}
