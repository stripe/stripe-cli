package profile

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"
)

// ConfigureProfile creates a profile when logging in
func (p *Profile) ConfigureProfile(apiKey string) error {
	runtimeViper, removeErr := removeKey(viper.GetViper(), "secret_key")
	if removeErr != nil {
		return removeErr
	}

	writeErr := p.writeConfig(runtimeViper, apiKey)
	if writeErr != nil {
		return writeErr
	}

	fmt.Println("You're configured and all set to get started")

	return nil
}

func (p *Profile) writeConfig(runtimeViper *viper.Viper, apiKey string) error {
	configFile := viper.ConfigFileUsed()

	err := makePath(configFile)
	if err != nil {
		return err
	}

	runtimeViper.SetConfigFile(configFile)

	// Ensure we preserve the config file type
	runtimeViper.SetConfigType(filepath.Ext(configFile))

	runtimeViper.Set(p.ProfileName+".device_name", strings.TrimSpace(p.DeviceName))
	runtimeViper.Set(p.ProfileName+".secret_key", strings.TrimSpace(apiKey))

	runtimeViper.MergeInConfig()
	runtimeViper.WriteConfig()

	return nil
}

// Temporary workaround until https://github.com/spf13/viper/pull/519 can remove a key from viper
func removeKey(v *viper.Viper, key string) (*viper.Viper, error) {
	configMap := v.AllSettings()

	delete(configMap, key)

	buf := new(bytes.Buffer)
	encodeErr := toml.NewEncoder(buf).Encode(configMap)
	if encodeErr != nil {
		return nil, encodeErr
	}

	nv := viper.New()
	nv.SetConfigType("toml") // hint to viper that we've encoded the data as toml

	err := nv.ReadConfig(buf)
	if err != nil {
		return nil, err
	}

	return nv, nil
}

func makePath(path string) error {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
