package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// Config handles all overall configuration for the CLI
type Config struct {
	Color        string
	LogLevel     string
	Profile      Profile
	ProfilesFile string
}

// GetProfilesFolder retrieves the folder where the profiles file is stored
// It searches for the xdg environment path first and will secondarily
// place it in the home directory
func (c *Config) GetProfilesFolder(xdgPath string) string {
	profilesPath := xdgPath

	log.WithFields(log.Fields{
		"prefix": "config.Config.GetProfilesFolder",
		"path":   profilesPath,
	}).Debug("Using profiles file")

	if profilesPath == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		profilesPath = filepath.Join(home, ".config")
	}

	return filepath.Join(profilesPath, "stripe")
}

// InitConfig reads in profiles file and ENV variables if set.
func (c *Config) InitConfig() {
	logFormatter := &prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}

	switch c.Color {
	case "on":
		ansi.ForceColors = true
		logFormatter.ForceColors = true
	case "off":
		ansi.DisableColors = true
		logFormatter.DisableColors = true
	case "auto":
		// Nothing to do
	default:
		log.Fatalf("Unrecognized color value: %s. Expected one of on, off, auto.", c.Color)
	}

	log.SetFormatter(logFormatter)

	// Set log level
	switch c.LogLevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Fatalf("Unrecognized log level value: %s. Expected one of debug, info, warn, error.", c.LogLevel)
	}

	if c.ProfilesFile != "" {
		// Use profiles file from the flag.
		viper.SetConfigFile(c.ProfilesFile)
	} else {
		profilesFolder := c.GetProfilesFolder(os.Getenv("XDG_CONFIG_HOME"))
		profilesFile := filepath.Join(profilesFolder, "config.toml")
		viper.SetConfigType("toml")
		viper.SetConfigFile(profilesFile)
	}

	// If a profiles file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.WithFields(log.Fields{
			"prefix": "config.Config.InitConfig",
			"path":   viper.ConfigFileUsed(),
		}).Debug("Using profiles file")
	}

	if c.Profile.DeviceName == "" {
		deviceName, err := os.Hostname()
		if err != nil {
			deviceName = "unknown"
		}
		c.Profile.DeviceName = deviceName
	}
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
