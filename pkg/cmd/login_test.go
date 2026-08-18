package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

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
			t.Cleanup(func() {
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
