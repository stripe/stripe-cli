//go:build wasm
// +build wasm

package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// ColorOn represnets the on-state for colors
const ColorOn = "on"

// ColorOff represents the off-state for colors
const ColorOff = "off"

// ColorAuto represents the auto-state for colors
const ColorAuto = "auto"

// IConfig allows us to add more implementations, such as ones for unit tests
type IConfig interface {
	GetProfile() *Profile
	GetConfigFolder(xdgPath string) string
	InitConfig()
	EditConfig() error
	PrintConfig() error
	RemoveProfile(profileName string) error
	RemoveAllProfiles() error
	WriteConfigField(field string, value interface{}) error
	GetInstalledPlugins() []string
}

// Config handles all overall configuration for the CLI
type Config struct {
	Color            string
	LogLevel         string
	Profile          Profile
	ProfilesFile     string
	InstalledPlugins []string
}

// GetProfile returns the Profile of the config
func (c *Config) GetProfile() *Profile {
	return &c.Profile
}

// GetConfigFolder retrieves the folder where the profiles file is stored
// It searches for the xdg environment path first and will secondarily
// place it in the home directory
func (c *Config) GetConfigFolder(xdgPath string) string {
	configPath := xdgPath

	if configPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		configPath = filepath.Join(home, ".config")
	}

	stripeConfigPath := filepath.Join(configPath, "stripe")

	log.WithFields(log.Fields{
		"prefix": "config.Config.GetProfilesFolder",
		"path":   stripeConfigPath,
	}).Debug("Using profiles file")

	return stripeConfigPath
}

// InitConfig reads in profiles file and ENV variables if set.
func (c *Config) InitConfig() {
	return
}

// EditConfig opens the configuration file in the default editor.
func (c *Config) EditConfig() error {
	return fmt.Errorf("unsupported platform")
}

// PrintConfig outputs the contents of the configuration file.
func (c *Config) PrintConfig() error {
	return fmt.Errorf("unsupported platform")
}

// GetInstalledPlugins returns a list of locally installed plugins.
// This does not vary by profile
func (c *Config) GetInstalledPlugins() []string {
	return []string{}
}

// RemoveProfile removes the profile whose name matches the provided
// profileName from the config file.
func (c *Config) RemoveProfile(profileName string) error {
	return fmt.Errorf("unsupported platform")
}

// RemoveAllProfiles removes all the profiles from the config file.
func (c *Config) RemoveAllProfiles() error {
	return fmt.Errorf("unsupported platform")
}

// isProfile identifies whether a value in the config pertains to a profile.
func isProfile(value interface{}) bool {
	return false
}

// WriteConfigField updates a configuration field and writes the updated
// configuration to disk.
func (c *Config) WriteConfigField(field string, value interface{}) error {
	return fmt.Errorf("unsupported platform")
}

// syncConfig merges a runtimeViper instance with the config file being used.
func syncConfig(runtimeViper *viper.Viper) error {
	return fmt.Errorf("unsupported platform")
}

// Temporary workaround until https://github.com/spf13/viper/pull/519 can remove a key from viper
func removeKey(v *viper.Viper, key string) (*viper.Viper, error) {
	configMap := v.AllSettings()
	path := strings.Split(key, ".")
	lastKey := strings.ToLower(path[len(path)-1])
	deepestMap := deepSearch(configMap, path[0:len(path)-1])
	delete(deepestMap, lastKey)

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

// taken from https://github.com/spf13/viper/blob/master/util.go#L199,
// we need this to delete configs, remove when viper supprts unset natively
func deepSearch(m map[string]interface{}, path []string) map[string]interface{} {
	for _, k := range path {
		m2, ok := m[k]
		if !ok {
			// intermediate key does not exist
			// => create it and continue from there
			m3 := make(map[string]interface{})
			m[k] = m3
			m = m3

			continue
		}

		m3, ok := m2.(map[string]interface{})
		if !ok {
			// intermediate key is a value
			// => replace with a new map
			m3 = make(map[string]interface{})
			m[k] = m3
		}

		// continue search from here
		m = m3
	}

	return m
}
