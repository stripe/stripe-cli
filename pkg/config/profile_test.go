package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestWriteProfile(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		DeviceName:  "st-testing",
		ProfileName: "tests",
		SecretKey: "sk_test_123",
	}

	c := &Config{
		Color: "auto",
		LogLevel: "info",
		Profile: p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()

	err := p.writeProfile(v)
	assert.NoError(t, err)

	assert.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expectedConfig := `
[tests]
  device_name = "st-testing"
  secret_key = "sk_test_123"
`
	assert.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(c.ProfilesFile)
}

func TestWriteProfilesMerge(t *testing.T) {
	profilesFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := Profile{
		ProfileName: "tests",
		DeviceName:  "st-testing",
		SecretKey: "sk_test_123",
	}

	c := &Config{
		Color: "auto",
		LogLevel: "info",
		Profile: p,
		ProfilesFile: profilesFile,
	}
	c.InitConfig()

	v := viper.New()
	writeErr := p.writeProfile(v)

	assert.NoError(t, writeErr)
	assert.FileExists(t, c.ProfilesFile)

	p.ProfileName = "tests-merge"
	writeErrTwo := p.writeProfile(v)
	assert.NoError(t, writeErrTwo)
	assert.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expectedConfig := `
[tests]
  device_name = "st-testing"
  secret_key = "sk_test_123"

[tests-merge]
  device_name = "st-testing"
  secret_key = "sk_test_123"
`

	assert.EqualValues(t, expectedConfig, string(configValues))

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
