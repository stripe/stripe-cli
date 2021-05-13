package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

// ColorOn represnets the on-state for colors
const ColorOn = "on"

// ColorOff represents the off-state for colors
const ColorOff = "off"

// ColorAuto represents the auto-state for colors
const ColorAuto = "auto"

// Config handles all overall configuration for the CLI
type Config struct {
	Color        string
	LogLevel     string
	Profile      Profile
	ProfilesFile string
}

// GetConfigFolder retrieves the folder where the profiles file is stored
// It searches for the xdg environment path first and will secondarily
// place it in the home directory
func (c *Config) GetConfigFolder(xdgPath string) string {
	configPath := xdgPath

	log.WithFields(log.Fields{
		"prefix": "config.Config.GetProfilesFolder",
		"path":   configPath,
	}).Debug("Using profiles file")

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

// InitConfig reads in profiles file and ENV variables if set.
func (c *Config) InitConfig() {
	logFormatter := &prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}

	if c.ProfilesFile != "" {
		viper.SetConfigFile(c.ProfilesFile)
	} else {
		configFolder := c.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
		configFile := filepath.Join(configFolder, "config.toml")
		c.ProfilesFile = configFile
		viper.SetConfigType("toml")
		viper.SetConfigFile(configFile)
		viper.SetConfigPermissions(os.FileMode(0600))

		// Try to change permissions manually, because we used to create files
		// with default permissions (0644)
		err := os.Chmod(configFile, os.FileMode(0600))
		if err != nil && !os.IsNotExist(err) {
			log.Fatalf("%s", err)
		}
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

	color, err := c.Profile.GetColor()
	if err != nil {
		log.Fatalf("%s", err)
	}

	switch color {
	case ColorOn:
		ansi.ForceColors = true
		logFormatter.ForceColors = true
	case ColorOff:
		ansi.DisableColors = true
		logFormatter.DisableColors = true
	case ColorAuto:
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
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.Fatalf("Unrecognized log level value: %s. Expected one of debug, info, warn, error.", c.LogLevel)
	}
}

// EditConfig opens the configuration file in the default editor.
func (c *Config) EditConfig() error {
	var err error

	fmt.Println("Opening config file:", c.ProfilesFile)

	switch runtime.GOOS {
	case "darwin", "linux":
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		cmd := exec.Command(editor, c.ProfilesFile)
		// Some editors detect whether they have control of stdin/out and will
		// fail if they do not.
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		return cmd.Run()
	case "windows":
		// As far as I can tell, Windows doesn't have an easily accesible or
		// comparable option to $EDITOR, so default to notepad for now
		err = exec.Command("notepad", c.ProfilesFile).Run()
	default:
		err = fmt.Errorf("unsupported platform")
	}

	return err
}

// PrintConfig outputs the contents of the configuration file.
func (c *Config) PrintConfig() error {
	if c.Profile.ProfileName == "default" {
		configFile, err := ioutil.ReadFile(c.ProfilesFile)
		if err != nil {
			return err
		}

		fmt.Print(string(configFile))
	} else {
		configs := viper.GetStringMapString(c.Profile.ProfileName)

		if len(configs) > 0 {
			fmt.Printf("[%s]\n", c.Profile.ProfileName)
			for field, value := range configs {
				fmt.Printf("  %s=%s\n", field, value)
			}
		}
	}

	return nil
}

// RemoveProfile removes the profile whose name matches the provided
// profileName from the config file.
func (c *Config) RemoveProfile(profileName string) error {
	runtimeViper := viper.GetViper()
	var err error

	for field, value := range runtimeViper.AllSettings() {
		if isProfile(value) && field == profileName {
			runtimeViper, err = removeKey(runtimeViper, field)
			if err != nil {
				return err
			}
		}
	}

	return syncConfig(runtimeViper)
}

// RemoveAllProfiles removes all the profiles from the config file.
func (c *Config) RemoveAllProfiles() error {
	runtimeViper := viper.GetViper()
	var err error

	for field, value := range runtimeViper.AllSettings() {
		if isProfile(value) {
			runtimeViper, err = removeKey(runtimeViper, field)
			if err != nil {
				return err
			}
		}
	}

	return syncConfig(runtimeViper)
}

// isProfile identifies whether a value in the config pertains to a profile.
func isProfile(value interface{}) bool {
	// TODO: ianjabour - ideally find a better way to identify projects in config
	_, ok := value.(map[string]interface{})
	return ok
}

// syncConfig merges a runtimeViper instance with the config file being used.
func syncConfig(runtimeViper *viper.Viper) error {
	runtimeViper.MergeInConfig()
	profilesFile := viper.ConfigFileUsed()
	runtimeViper.SetConfigFile(profilesFile)
	// Ensure we preserve the config file type
	runtimeViper.SetConfigType(filepath.Ext(profilesFile))

	err := runtimeViper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
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
