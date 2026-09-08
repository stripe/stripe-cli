package cmd

import (
	"context"
	"os"
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
//
// revokeToken and initiateLogin are stubbed rather than backed by an
// httptest server: --access-base is validated against a hardcoded
// production/QA allowlist before any of this logic runs (it carries OAuth
// credentials), so a mock server URL can never be assigned to it.
func TestLoginNewSessionRevokesPreviousToken(t *testing.T) {
	origRevokeToken, origInitiateLogin := revokeToken, initiateLogin
	t.Cleanup(func() {
		revokeToken = origRevokeToken
		initiateLogin = origInitiateLogin
	})

	var revokeCalled, initiateLoginCalled bool
	revokeToken = func(ctx context.Context, accessBaseURL string) error {
		revokeCalled = true
		rt, err := config.KeyRing.Get(config.OAuthRefreshTokenKeychainKey)
		require.NoError(t, err)
		assert.Equal(t, "oart_previous_refresh", string(rt))
		return nil
	}
	initiateLogin = func(ctx context.Context, dashboardBaseURL, accessBaseURL string, cfg *config.Config) error {
		initiateLoginCalled = true
		assert.True(t, revokeCalled, "expected the previous session to be revoked before continuing")
		return nil
	}

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
	lc.cmd.SetContext(context.Background())

	require.NoError(t, lc.runLoginCmd(lc.cmd, []string{}))
	assert.True(t, revokeCalled, "expected --new-session to revoke the previous OAuth token")
	assert.True(t, initiateLoginCalled, "expected login to continue after revoking")
}

// The profile name is checked before authentication starts, so the user is not
// sent through a browser flow whose credentials could not be saved afterwards.
func TestLoginValidatesProfileNameBeforeAuthentication(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		config      string
		wantError   string
	}{
		{
			name:        "new dotted profile is rejected",
			profileName: "example.project",
			config:      "[default]\ndisplay_name = 'Default'\n",
			wantError:   `profile name "example.project" cannot contain a period; use a hyphen or underscore instead`,
		},
		{
			name:        "existing dotted profile is allowed through",
			profileName: "example.project",
			config:      "[\"example.project\"]\ndisplay_name = 'Existing profile'\n",
		},
		{
			name:        "normal profile is allowed through",
			profileName: "example-project",
			config:      "[default]\ndisplay_name = 'Default'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Config is package state, and viper.Reset below drops the pflag
			// binding that ReBindKeys would otherwise use to restore the
			// default profile name. Put it back by hand so later tests in this
			// package don't observe this test's profile.
			origConfig := Config
			t.Cleanup(func() {
				Config = origConfig
				config.KeyRing = nil
				viper.Reset()
			})

			profilesFile := filepath.Join(t.TempDir(), "config.toml")
			require.NoError(t, os.WriteFile(profilesFile, []byte(tt.config), 0600))
			Config = config.Config{
				LogLevel: "info",
				Profile: config.Profile{
					ProfileName: tt.profileName,
					DeviceName:  "test-device",
				},
				ProfilesFile: profilesFile,
			}
			config.KeyRing = keyring.NewMemoryStore(nil)
			Config.InitConfig()

			lc := newLoginCmd()
			lc.cmd.SetContext(context.Background())
			// An invalid name must fail before the dashboard URL is even looked at,
			// so an unreachable one is used to prove nothing downstream ran.
			lc.dashboardBaseURL = "not-a-valid-url"
			err := lc.runLoginCmd(lc.cmd, nil)

			if tt.wantError == "" {
				// The name passed; execution reached the next check rather than
				// being rejected for the profile name.
				require.Error(t, err)
				require.NotContains(t, err.Error(), "profile name")
				return
			}

			require.EqualError(t, err, tt.wantError)
		})
	}
}
