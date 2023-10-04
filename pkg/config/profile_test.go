package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestWriteProfile(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		DeviceName:     "st-testing",
		ProfileName:    "tests",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()

	fmt.Println(profilesFile)

	err := p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expiresAt := getKeyExpiresAt()
	expectedConfig := `[tests]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'

`

	require.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(c.ProfilesFile)
}

func TestWriteProfilesMerge(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName:    "tests",
		DeviceName:     "st-testing",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()
	writeErr := p.writeProfile(v)

	require.NoError(t, writeErr)
	require.FileExists(t, c.ProfilesFile)

	p.ProfileName = "tests-merge"
	writeErrTwo := p.writeProfile(v)
	require.NoError(t, writeErrTwo)
	require.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expiresAt := getKeyExpiresAt()
	expectedConfig := `[tests]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'

[tests-merge]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'

`

	require.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(c.ProfilesFile)
}

func TestExperimentalFields(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName:    "tests",
		DeviceName:     "st-testing",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	require.NoError(t, err)

	experimentalFields := p.GetExperimentalFields()
	require.Equal(t, "", experimentalFields.ContextualName)
	require.Equal(t, "", experimentalFields.StripeHeaders)
	require.Equal(t, "", experimentalFields.PrivateKey)

	p.WriteConfigField("experimental.stripe_headers", "test-headers")
	p.WriteConfigField("experimental.contextual_name", "test-name")
	p.WriteConfigField("experimental.private_key", "test-key")

	experimentalFields = p.GetExperimentalFields()
	require.Equal(t, "test-name", experimentalFields.ContextualName)
	require.Equal(t, "test-headers", experimentalFields.StripeHeaders)
	require.Equal(t, "test-key", experimentalFields.PrivateKey)

	cleanUp(c.ProfilesFile)
}

func TestOldProfileDeleted(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName:    "test",
		DeviceName:     "device-before-test",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "display-name-before-test",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	p.WriteConfigField("experimental.stripe_headers", "test-headers")

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	untouchedProfile := Profile{
		ProfileName:    "foo",
		DeviceName:     "foo-device-name",
		TestModeAPIKey: "foo_test_123",
	}
	err = untouchedProfile.writeProfile(v)
	require.NoError(t, err)

	p = Profile{
		ProfileName:    "test",
		DeviceName:     "device-after-test",
		TestModeAPIKey: "sk_test_456",
		DisplayName:    "",
	}

	v = p.deleteProfile(v)
	err = p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	// Overwrites keys
	require.Equal(t, "device-after-test", v.GetString(p.GetConfigField(DeviceNameName)))
	require.Equal(t, "sk_test_456", v.GetString(p.GetConfigField(TestModeAPIKeyName)))
	require.Equal(t, "", v.GetString(p.GetConfigField(DisplayNameName)))
	// Deletes nested keys
	require.False(t, v.IsSet(v.GetString(p.GetConfigField("experimental.stripe_headers"))))
	require.False(t, v.IsSet(v.GetString(p.GetConfigField("experimental"))))
	// Leaves the other profile untouched
	require.Equal(t, "foo-device-name", v.GetString(untouchedProfile.GetConfigField(DeviceNameName)))
	require.Equal(t, "foo_test_123", v.GetString(untouchedProfile.GetConfigField(TestModeAPIKeyName)))

	cleanUp(c.ProfilesFile)
}

func TestExperimentalFieldsEmptyWhenAPIKeyIsOverridden(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName:    "tests",
		DeviceName:     "st-testing",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	require.NoError(t, err)

	p.WriteConfigField("experimental.stripe_headers", "test-headers")
	p.WriteConfigField("experimental.contextual_name", "test-name")
	p.WriteConfigField("experimental.private_key", "test-key")

	os.Setenv("STRIPE_API_KEY", "from-env")
	defer os.Unsetenv("STRIPE_API_KEY")

	experimentalFields := p.GetExperimentalFields()
	require.Equal(t, "", experimentalFields.ContextualName)
	require.Equal(t, "", experimentalFields.StripeHeaders)
	require.Equal(t, "", experimentalFields.PrivateKey)

	cleanUp(c.ProfilesFile)
}

func helperLoadBytes(t *testing.T, name string) []byte {
	bytes, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}

func cleanUp(file string) {
	os.Remove(file)
}
