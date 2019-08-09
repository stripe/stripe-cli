package config

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// Profile handles all things related to managing the project specific configurations
type Profile struct {
	DeviceName  string
	ProfileName string
	APIKey      string
}

// CreateProfile creates a profile when logging in
func (p *Profile) CreateProfile() error {
	writeErr := p.writeProfile(viper.GetViper())
	if writeErr != nil {
		return writeErr
	}

	return nil
}

// GetDeviceName returns the configured device name
func (p *Profile) GetDeviceName() (string, error) {
	deviceName := viper.GetString("device_name")
	if deviceName != "" {
		return deviceName, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString(p.GetConfigField("device_name")), nil
	}

	return "", errors.New("your device name has not been configured. Use `stripe login` to set your device name")
}

// GetAPIKey will return the existing key for the given profile
func (p *Profile) GetAPIKey() (string, error) {
	// If the user doesn't have an api_key field set, they might be using an
	// old configuration so try to read from secret_key
	if !viper.IsSet(p.GetConfigField("api_key")) {
		p.RegisterAlias("api_key", "secret_key")
	}

	key := viper.GetString("api_key")
	if key != "" {
		err := validators.APIKey(key)
		if err != nil {
			return "", err
		}
		return key, nil
	}

	// Try to fetch the API key from the configuration file
	if err := viper.ReadInConfig(); err == nil {
		key := viper.GetString(p.GetConfigField("api_key"))
		err := validators.APIKey(key)
		if err != nil {
			return "", err
		}
		return key, nil
	}

	return "", errors.New("your API key has not been configured. Use `stripe login` to set your API key")
}

// GetConfigField returns the configuration field for the specific profile
func (p *Profile) GetConfigField(field string) string {
	return p.ProfileName + "." + field
}

func (p *Profile) RegisterAlias(alias, key string) {
	viper.RegisterAlias(p.GetConfigField(alias), p.GetConfigField(key))
}

func (p *Profile) WriteConfigField(field, value string) error {
	viper.Set(p.GetConfigField(field), value)
	return viper.WriteConfig()
}

func (p *Profile) DeleteConfigField(field string) error {
	v, err := removeKey(viper.GetViper(), p.GetConfigField(field))
	if err != nil {
		return err
	}
	return p.writeProfile(v)
}

func (p *Profile) writeProfile(runtimeViper *viper.Viper) error {
	profilesFile := viper.ConfigFileUsed()

	err := makePath(profilesFile)
	if err != nil {
		return err
	}

	runtimeViper.SetConfigFile(profilesFile)

	if p.DeviceName != "" {
		runtimeViper.Set(p.GetConfigField("device_name"), strings.TrimSpace(p.DeviceName))
	}
	if p.APIKey != "" {
		runtimeViper.Set(p.GetConfigField("api_key"), strings.TrimSpace(p.APIKey))
	}

	// Ensure we preserve the config file type
	runtimeViper.SetConfigType(filepath.Ext(profilesFile))

	runtimeViper.MergeInConfig()
	runtimeViper.WriteConfig()

	return nil
}
