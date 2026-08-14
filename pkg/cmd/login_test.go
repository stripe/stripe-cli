package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// TestLoginNewSessionRevokesPreviousToken verifies that `stripe login --new-session`
// revokes the previously stored OAuth token, same as `stripe logout`, before
// starting a new login flow.
func TestLoginNewSessionRevokesPreviousToken(t *testing.T) {
	var revokeCalled bool

	accessSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/stripecli/oauth2/revoke":
			revokeCalled = true
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "oart_previous_refresh", r.FormValue("token"))
			w.WriteHeader(http.StatusOK)
		case "/stripecli/oauth2/device/authorization":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"device_code":"dc","user_code":"uc","verification_uri":"https://dashboard.stripe.com/verify","expires_in":600,"interval":5}`))
		default:
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
	}))
	defer accessSrv.Close()

	dashboardSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A 3xx response from /stripecli/auth signals that the OAuth device-code
		// flow (rather than the legacy RAK flow) should be used.
		w.Header().Set("Location", "https://dashboard.stripe.com/")
		w.WriteHeader(http.StatusFound)
	}))
	defer dashboardSrv.Close()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	Config = config.Config{
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "default",
			DeviceName:  "test-device",
		},
		ProfilesFile: profilesFile,
	}
	Config.InitConfig()

	config.KeyRing = keyring.NewMemoryStore(map[string][]byte{
		config.UATKeychainItemKey:           []byte("oak_previous_uat"),
		config.OAuthRefreshTokenKeychainKey: []byte("oart_previous_refresh"),
	})
	t.Cleanup(func() {
		config.KeyRing = nil
		viper.Reset()
	})

	lc := newLoginCmd()
	lc.newSession = true
	lc.nonInteractive = true
	lc.dashboardBaseURL = dashboardSrv.URL
	lc.accessBaseURL = accessSrv.URL
	lc.cmd.SetContext(context.Background())

	require.NoError(t, lc.runLoginCmd(lc.cmd, []string{}))
	assert.True(t, revokeCalled, "expected --new-session to revoke the previous OAuth token")
}
