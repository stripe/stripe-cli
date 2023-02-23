package keys

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestSaveLoginDetails(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "stripe", "config.toml")
	c := &config.Config{
		LogLevel: "info",
		Profile: config.Profile{
			ProfileName: "tests",
		},
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	configurer := NewRAKConfigurer(c, afero.NewOsFs())
	err := configurer.SaveLoginDetails(&PollAPIKeyResponse{
		Redeemed:               true,
		AccountID:              "acct_123",
		AccountDisplayName:     "",
		LiveModeAPIKey:         "rk_live_1234567890000",
		TestModeAPIKey:         "rk_test_1234567890000",
		LiveModePublishableKey: "pk_live_1234567890000",
		TestModePublishableKey: "pk_test_1234567890000",
	})
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(profilesFile)

	err = v.ReadInConfig()
	require.NoError(t, err)

	assert.Equal(t, "acct_123", v.GetString("tests.account_id"))
	assert.Equal(t, "", v.GetString("tests.display_name"))
	assert.Equal(t, "rk_live_*********0000", v.GetString("tests.live_mode_api_key"))
	assert.Equal(t, "pk_live_1234567890000", v.GetString("tests.live_mode_pub_key"))
	assert.Equal(t, "rk_test_1234567890000", v.GetString("tests.test_mode_api_key"))
	assert.Equal(t, "pk_test_1234567890000", v.GetString("tests.test_mode_pub_key"))
}
