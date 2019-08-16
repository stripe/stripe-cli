package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestWriteProfile(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		DeviceName:  "st-testing",
		ProfileName: "tests",
		APIKey:      "sk_test_123",
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
	expectedConfig := `
[tests]
  api_key = "sk_test_123"
  device_name = "st-testing"
`
	require.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(c.ProfilesFile)
}

func TestWriteProfilesMerge(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName: "tests",
		DeviceName:  "st-testing",
		APIKey:      "sk_test_123",
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
	expectedConfig := `
[tests]
  api_key = "sk_test_123"
  device_name = "st-testing"

[tests-merge]
  api_key = "sk_test_123"
  device_name = "st-testing"
`

	require.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(c.ProfilesFile)
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
