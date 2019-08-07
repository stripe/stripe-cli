package config

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// Profile handles all things related to managing the project specific configurations
type Profile struct {
	DeviceName  string
	ProfileName string
	SecretKey   string
}

// CreateProfile creates a profile when logging in
func (p *Profile) CreateProfile() error {
	runtimeViper, removeErr := removeKey(viper.GetViper(), "secret_key")
	if removeErr != nil {
		return removeErr
	}

	writeErr := p.writeProfile(runtimeViper)
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

// GetSecretKey will return the existing key for the given profile
func (p *Profile) GetSecretKey() (string, error) {
	// Try to fetch the API key from the command-line flag or the environment first
	key := viper.GetString("secret_key")
	if key != "" {
		err := validators.APIKey(key)
		if err != nil {
			return "", err
		}
		return key, nil
	}

	// Try to fetch the API key from the configuration file
	if err := viper.ReadInConfig(); err == nil {
		key := viper.GetString(p.GetConfigField("secret_key"))
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

func (p *Profile) PrintConfig() {
	var projects []string
	allSettings := viper.AllSettings()
	delete(allSettings, "secret_key")

	if p.ProfileName == "default" {
		projects = []string{"default"}
		for project := range allSettings {
			if project != "default" {
				projects = append(projects, project)
			}
		}
	} else {
		projects = []string{p.ProfileName}
	}

	for _, project := range projects {
		fmt.Println(fmt.Sprintf("[%s]", project))
		projectSettings := allSettings[project].(map[string]interface{})
		for field, value := range projectSettings {
			fmt.Println(fmt.Sprintf("%s=%s", field, value))
		}

		fmt.Println()
	}
}

func (p *Profile) writeProfile(runtimeViper *viper.Viper) error {
	profilesFile := viper.ConfigFileUsed()

	err := makePath(profilesFile)
	if err != nil {
		return err
	}

	runtimeViper.SetConfigFile(profilesFile)

	// Ensure we preserve the config file type
	runtimeViper.SetConfigType(filepath.Ext(profilesFile))

	runtimeViper.MergeInConfig()
	runtimeViper.WriteConfig()

	return nil
}
