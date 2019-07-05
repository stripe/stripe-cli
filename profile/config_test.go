package profile

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)


func TestRemoveKey(t *testing.T) {
	v := viper.New()
	v.Set("remove", "me")
	v.Set("stay", "here")

	nv, err := removeKey(v, "remove")
	assert.NoError(t, err)

	assert.EqualValues(t, []string{"stay"}, nv.AllKeys())
	assert.ElementsMatch(t, []string{"stay", "remove"}, v.AllKeys())
}


func TestWriteConfig(t *testing.T) {
	configFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := &Profile{
		Color:       "auto",
		ConfigFile:  configFile,
		LogLevel:  "info",
		ProfileName: "tests",
		DeviceName: "st-testing",
	}

	p.InitConfig()

	apiKey := "sk_test_123"
	v := viper.New()

	err := p.writeConfig(v, apiKey)
	assert.NoError(t, err)

	assert.FileExists(t, p.ConfigFile)

	configValues := helperLoadBytes(t, p.ConfigFile)
	expectedConfig := `
[tests]
  device_name = "st-testing"
  secret_key = "sk_test_123"
`
	assert.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(p.ConfigFile)
}


func TestWriteConfigMerge(t *testing.T) {
	configFile := filepath.Join(os.TempDir(), "stripe", "config.toml")
	p := &Profile{
		Color:       "auto",
		ConfigFile:  configFile,
		LogLevel:  "info",
		ProfileName: "tests",
		DeviceName: "st-testing",
	}
	p.InitConfig()
	v := viper.New()
	writeErr := writeFile(v, p)
	assert.NoError(t, writeErr)
	assert.FileExists(t, p.ConfigFile)

	p.ProfileName = "tests-merge"
	writeErrTwo := writeFile(v, p)
	assert.NoError(t, writeErrTwo)
	assert.FileExists(t, p.ConfigFile)

	configValues := helperLoadBytes(t, p.ConfigFile)
	expectedConfig := `
[tests]
  device_name = "st-testing"
  secret_key = "sk_test_123"

[tests-merge]
  device_name = "st-testing"
  secret_key = "sk_test_123"
`

	assert.EqualValues(t, expectedConfig, string(configValues))

	cleanUp(p.ConfigFile)

}

func writeFile(v *viper.Viper, p *Profile) error {
	apiKey := "sk_test_123"

	err := p.writeConfig(v, apiKey)

	return err

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
