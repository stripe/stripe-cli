package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/stripe/stripe-cli/ansi"
	"github.com/stripe/stripe-cli/validators"
)

// Profile handles all things related to managing the project specific configurations
type Profile struct {
	Color       string
	ConfigFile  string
	LogLevel    string
	ProfileName string
	DeviceName string
}

// GetDeviceName returns the configured device name
func (p *Profile) GetDeviceName() (string, error) {
	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString("default.device_name"), nil
	}

	return "", errors.New("your device name has not been configured. Use `stripe configure` to set your device name")
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

	return "", errors.New("your API key has not been configured. Use `stripe configure` to set your API key")
}

// GetConfigFolder retrieves the folder where the config file is stored
// It searches for the xdg environment path first and will secondarily
// place it in the home directory
func (p *Profile) GetConfigFolder(xdgPath string) string {
	configPath := xdgPath

	log.WithFields(log.Fields{
		"prefix": "profile.Profile.GetConfigFolder",
		"path":   configPath,
	}).Debug("Using config file")

	if configPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		configPath = filepath.Join(home, ".config")
	}

	return filepath.Join(configPath, "stripe")
}

// GetConfigField returns the configuration field for the specific profile
func (p *Profile) GetConfigField(field string) string {
	return p.ProfileName + "." + field
}

// InitConfig reads in config file and ENV variables if set.
func (p *Profile) InitConfig() {
	logFormatter := &prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}

	switch p.Color {
	case "on":
		ansi.ForceColors = true
		logFormatter.ForceColors = true
	case "off":
		ansi.DisableColors = true
		logFormatter.DisableColors = true
	case "auto":
		// Nothing to do
	default:
		log.Fatalf("Unrecognized color value: %s. Expected one of on, off, auto.", p.Color)
	}

	log.SetFormatter(logFormatter)

	// Set log level
	switch p.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Fatalf("Unrecognized log level value: %s. Expected one of debug, info, warn, error.", p.LogLevel)
	}

	if p.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(p.ConfigFile)
	} else {
		viper.SetConfigType("toml")
		// Search config in home directory or xdg path with name "config.toml".
		viper.AddConfigPath(p.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME")))
		// TODO(tomer) - support overriding with configs in local dir
		viper.SetConfigName("config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.WithFields(log.Fields{
			"prefix": "profile.Profile.InitConfig",
			"path":   viper.ConfigFileUsed(),
		}).Debug("Using config file")
	}

	if p.DeviceName == "" {
		deviceName, err := os.Hostname()
		if err != nil {
			deviceName = "unknown"
		}
		p.DeviceName = deviceName
	}
}
