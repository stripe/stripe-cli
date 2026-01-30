package config

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/BurntSushi/toml"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/git"
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
	CopyProfile(source string, target string) error
	ListProfiles() error
	SwitchProfile(targetProfileName string) error
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
	logFormatter := &prefixed.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
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

	if os.Getenv("STRIPE_CLI_CANARY") == "true" {
		log.WithFields(log.Fields{
			"prefix": "config.Config.InitConfig",
		}).Debug("Running with STRIPE_CLI_CANARY=true")
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

	// initialize key ring
	KeyRing, _ = keyring.Open(keyring.Config{
		ServiceName: KeyManagementService,
	})

	// redact livemode values for existing configs
	c.Profile.redactAllLivemodeValues()
}

// EditConfig opens the configuration file in the default editor.
func (c *Config) EditConfig() error {
	fmt.Println("Opening config file:", c.ProfilesFile)

	editor, err := git.NewEditor(c.ProfilesFile)
	if err != nil {
		return err
	}

	_, err = editor.EditContent()
	return err
}

func (c *Config) CopyProfile(source string, target string) error {

	if source == "" {
		return fmt.Errorf("source profile name cannot be empty")
	}
	if target == "" {
		return fmt.Errorf("target profile name cannot be empty")
	}

	if source == target {
		return fmt.Errorf("cannot copy profile to itself")
	}

	runtimeViper := viper.GetViper()
	safeSource := strings.ReplaceAll(source, ".", " ")
	if !runtimeViper.IsSet(safeSource) {
		return fmt.Errorf("source profile '%s' does not exist", source)
	}
	existing := runtimeViper.Get(safeSource)
	if !isProfile(existing) {
		return fmt.Errorf("source '%s' is not a profile", source)
	}

	// Clone the profile map and update profile_name
	safeTarget := strings.ReplaceAll(target, ".", " ")
	existingMap := existing.(map[string]interface{})
	newProfile := maps.Clone(existingMap)
	newProfile["profile_name"] = safeTarget

	runtimeViper.Set(safeTarget, newProfile)

	return writeConfig(runtimeViper)
}

func (c *Config) ListProfiles() error {
	runtimeViper := viper.GetViper()
	var profiles []string

	for _, value := range runtimeViper.AllSettings() {
		// TODO: there's probably a better way to e.g. hydrate a Profile and read from there?
		profile, isProfile := value.(map[string]interface{})
		if isProfile {
			displayName, _ := profile["display_name"].(string)
			if !slices.Contains(profiles, displayName) {
				profiles = append(profiles, displayName)
			}
		}
	}

	// TODO: sort by most recently used
	sort.Strings(profiles)

	if len(profiles) == 0 {
		fmt.Println("No profiles found.")
	} else {
		fmt.Println("Available profiles:")
		for _, profile := range profiles {
			// GetDisplayName() reads from the config file to ensure consistency
			// with the display names we extracted from AllSettings() above
			if profile == c.Profile.GetDisplayName() {
				fmt.Printf("  * %s (active)\n", profile)
			} else {
				fmt.Printf("    %s\n", profile)
			}
		}
	}

	return nil
}

// PrintConfig outputs the contents of the configuration file.
func (c *Config) PrintConfig() error {
	profileName := c.Profile.ProfileName

	if profileName == "default" {
		configFile, err := os.ReadFile(c.ProfilesFile)
		if err != nil {
			return err
		}

		fmt.Print(string(configFile))
	} else {
		configs := viper.GetStringMapString(profileName)

		if len(configs) > 0 {
			fmt.Printf("[%s]\n", profileName)
			for field, value := range configs {
				fmt.Printf("  %s=%s\n", field, value)
			}
		}
	}

	return nil
}

// GetInstalledPlugins returns a list of locally installed plugins.
// This does not vary by profile
func (c *Config) GetInstalledPlugins() []string {
	runtimeViper := viper.GetViper()

	return runtimeViper.GetStringSlice("installed_plugins")
}

func (c *Config) SwitchProfile(profileName string) error {
	// First copy the active profile to a different key
	// TODO: should this be account id instead of display name?
	if err := c.CopyProfile("default", c.Profile.GetDisplayName()); err != nil {
		return err
	}

	// Then copy the target profile to "default"
	// This makes the target profile the active one
	// since the CLI always uses the "default" profile internally
	if err := c.CopyProfile(profileName, "default"); err != nil {
		return err
	}

	// Remove the old profile key since it's now been copied to "default"
	// This keeps the config file clean by not having duplicate data
	c.RemoveProfile(profileName)

	// Finally, reload the config to pick up the new "default" profile
	c.InitConfig()

	fmt.Printf("Switched to profile: %s\n", profileName)

	return nil
}

// RemoveProfile removes the profile whose name matches the provided
// profileName from the config file.
func (c *Config) RemoveProfile(profileName string) error {
	runtimeViper := viper.GetViper()
	var err error

	for field, value := range runtimeViper.AllSettings() {
		if isProfile(value) {
			var profileNameAttr string
			switch v := value.(type) {
			case map[string]interface{}:
				if pn, ok := v["profile_name"].(string); ok {
					profileNameAttr = pn
				}
			case map[string]string:
				profileNameAttr = v["profile_name"]
			}
			if field == profileName || profileNameAttr == profileName {
				runtimeViper, err = removeKey(runtimeViper, field)
				if err != nil {
					return err
				}

				deleteLivemodeKey(LiveModeAPIKeyName, field)
			}
		}
	}

	return writeConfig(runtimeViper)
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

			deleteLivemodeKey(LiveModeAPIKeyName, field)
		}
	}

	return writeConfig(runtimeViper)
}

func deleteLivemodeKey(key string, profile string) error {
	fieldID := profile + "." + key
	existingKeys, err := KeyRing.Keys()
	if err != nil {
		return err
	}
	for _, item := range existingKeys {
		if item == fieldID {
			KeyRing.Remove(fieldID)
			return nil
		}
	}
	return nil
}

// isProfile identifies whether a value in the config pertains to a profile.
func isProfile(value interface{}) bool {
	// TODO: ianjabour - ideally find a better way to identify projects in config
	_, ok := value.(map[string]interface{})
	if !ok {
		_, ok = value.(map[string]string)
	}

	return ok
}

// WriteConfigField updates a configuration field and writes the updated
// configuration to disk.
func (c *Config) WriteConfigField(field string, value interface{}) error {
	runtimeViper := viper.GetViper()
	runtimeViper.Set(field, value)

	return runtimeViper.WriteConfig()
}

// writeConfig writes a viper instance to the config file and syncs the global viper.
func writeConfig(runtimeViper *viper.Viper) error {
	profilesFile := viper.ConfigFileUsed()
	runtimeViper.SetConfigFile(profilesFile)
	configType := strings.TrimPrefix(filepath.Ext(profilesFile), ".")
	runtimeViper.SetConfigType(configType)

	if err := runtimeViper.WriteConfig(); err != nil {
		return err
	}

	// Reset global viper and re-read from file.
	// We must reset because ReadInConfig merges with existing values rather than
	// replacing them - so deleted keys would persist without this reset.
	viper.Reset()
	viper.SetConfigFile(profilesFile)
	viper.SetConfigType(configType)
	viper.SetConfigPermissions(os.FileMode(0600))

	return viper.ReadInConfig()
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
