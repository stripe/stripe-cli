package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
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

func TestAPIKeyLogLevel(t *testing.T) {
	// Set the level to debug
	logrus.SetLevel(logrus.DebugLevel)

	c := &Config{
		Color:    "auto",
		LogLevel: "debug",
		Profile: Profile{
			ProfileName:    "tests",
			TestModeAPIKey: "asdas",
		},
		ProfilesFile: "",
	}

	// For debug mode, the error should complain about a config file missing
	// since we did not init the config
	key, err := c.Profile.GetAPIKey(false)
	assert.EqualError(t, err, `Config File "config" Not Found in "[]"`)
	assert.Equal(t, "", key)

	// In info mode, it should give a cleaner error about the key not being
	// configured
	logrus.SetLevel(logrus.InfoLevel)
	key, err = c.Profile.GetAPIKey(false)
	assert.EqualError(t, err, "you have not configured API keys yet")
	assert.Equal(t, "", key)
}

func helperLoadBytes(t *testing.T, name string) []byte {
	bytes, err := ioutil.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}

func cleanUp(file string) {
	os.Remove(file)
}
