// Package config manages CLI configuration and profiles.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/fsutil"
	"github.com/stripe/stripe-cli/pkg/git"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// Top-level config.toml keys that belong to the CLI itself rather than to a
// profile. A profile with one of these names is a collision, which is what
// moving profiles under the reserved profiles table fixes.
const (
	// ColorName is the color setting. It is also a valid profile field, so it can
	// appear both at the top level and inside a profile.
	ColorName = "color"

	// InstalledPluginsKey lists the locally installed plugins.
	InstalledPluginsKey = "installed_plugins"

	// MachineUUIDKey is the persistent machine identifier used for telemetry.
	MachineUUIDKey = "machine_uuid"
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
	RemoveAuthFields(profileName string) error
	RemoveAllAuthFields() error
	WriteConfigField(field string, value interface{}) error
	GetInstalledPlugins() []string
	GetMachineUUID() string
}

// Config handles all overall configuration for the CLI
type Config struct {
	Color            string
	LogLevel         string
	Profile          Profile
	ProfilesFile     string
	InstalledPlugins []string

	// warnedConfigVersion keeps the unsupported-version warning to one line per
	// invocation. InitConfig runs more than once: root.go initializes it eagerly to
	// register plugins before cobra parses flags, cobra.OnInitialize runs it again,
	// and SwitchProfile reloads through it.
	warnedConfigVersion bool
}

// GetProfile returns the Profile of the config
func (c *Config) GetProfile() *Profile {
	return &c.Profile
}

// GetConfigFolder retrieves the folder where the profiles file is stored
// It searches for the xdg environment path first and will secondarily
// place it in the home directory

func (c *Config) GetConfigFolder(xdgPath string) string {
	return getConfigFolder(xdgPath)
}

func getConfigFolder(xdgPath string) string {
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

// CredentialsFilePath returns the path of the plain-text credentials file used
// by the file fallback store when the OS keyring is unavailable.
func CredentialsFilePath() string {
	return filepath.Join(getConfigFolder(os.Getenv("XDG_CONFIG_HOME")), "credentials.json")
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

		// Reads tolerate an unknown version: both layouts are tried, so a newer
		// file's profiles are usually still found. Warn rather than exit, so an
		// older pinned CLI keeps working for read-only commands. writeConfig is
		// where an unknown version is actually refused.
		if _, err := configVersion(viper.GetViper()); err != nil && !c.warnedConfigVersion {
			c.warnedConfigVersion = true
			log.Warnf("%s: %s", viper.ConfigFileUsed(), err)
		}
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

	// initialize secure credential store (tests may pre-set KeyRing to a mock)
	if KeyRing == nil {
		KeyRing = keyring.NewSecureStore(KeyManagementService, CredentialsFilePath())
	}

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
		return errorcategory.Errorf(errorcategory.UserInput, "source profile name cannot be empty")
	}
	if target == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "target profile name cannot be empty")
	}

	if source == target {
		return errorcategory.Errorf(errorcategory.UserInput, "cannot copy profile to itself")
	}

	runtimeViper := viper.GetViper()
	safeSource := strings.ReplaceAll(source, ".", " ")

	// Prefer the v2 table, where any entry is a profile, and fall back to the
	// top level, where a profile is only recognizable by its display_name.
	existing := runtimeViper.Get(ProfilesTableName + "." + safeSource)
	if _, ok := toStringMap(existing); !ok {
		if !runtimeViper.IsSet(safeSource) {
			return errorcategory.Errorf(errorcategory.UserInput, "source profile '%s' does not exist", source)
		}

		existing = runtimeViper.Get(safeSource)
		if !isProfile(existing) {
			return errorcategory.Errorf(errorcategory.UserInput, "source '%s' is not a profile", source)
		}
	}

	safeTarget := strings.ReplaceAll(target, ".", " ")
	existingMap, _ := toStringMap(existing)
	newProfile := make(map[string]interface{})
	for k, v := range existingMap {
		if isPluginConfigSection(v) {
			// Skip plugin config sections of the form <scope>.<plugin config key>.
			continue
		}
		newProfile[k] = v
	}
	newProfile["profile_name"] = safeTarget

	runtimeViper.Set(profileTableKeyForWrite(runtimeViper, safeTarget), newProfile)

	return writeConfig(runtimeViper)
}

func (c *Config) ListProfiles() error {
	runtimeViper := viper.GetViper()
	var profiles []string

	for _, entry := range listProfileEntries(runtimeViper) {
		displayName, _ := entry.settings["display_name"].(string)
		if displayName != "" && !slices.Contains(profiles, displayName) {
			profiles = append(profiles, displayName)
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
		configs := viper.GetStringMapString(ProfilesTableName + "." + profileName)
		if len(configs) == 0 {
			configs = viper.GetStringMapString(profileName)
		}

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

	return runtimeViper.GetStringSlice(InstalledPluginsKey)
}

// GetMachineUUID returns the persistent machine UUID from config,
// generating and saving one if it doesn't exist.
func (c *Config) GetMachineUUID() string {
	runtimeViper := viper.GetViper()
	id := runtimeViper.GetString(MachineUUIDKey)
	if id != "" {
		return id
	}
	id = uuid.NewString()
	_ = c.WriteConfigField(MachineUUIDKey, id)
	return id
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

// reservedTopLevelKeys are config keys that sit alongside profiles at the top
// level of the file but are not profiles. They are excluded from profile removal
// so that a name collision cannot destroy machine-wide settings — or, for
// "profiles", the entire v2 profile table.
var reservedTopLevelKeys = map[string]bool{
	ConfigVersionName:   true,
	ProfilesTableName:   true,
	ColorName:           true,
	InstalledPluginsKey: true,
	MachineUUIDKey:      true,
	PluginConfigsKey:    true,
	"project-name":      true,
	UserInfoName:        true,
}

// ErrProfileNotFound is returned when no profile matches the requested name.
var ErrProfileNotFound = errorcategory.New(errorcategory.UserInput, "profile not found")

// isReservedConfigKey reports whether name addresses machine-wide config rather
// than a profile.
//
// The first path segment is what matters: a name like "plugin_configs.apps"
// resolves to a real table, so without this it would look like a legacy profile
// whose name contains a period and be removable as one.
func isReservedConfigKey(name string) bool {
	first, _, _ := strings.Cut(strings.ToLower(name), ".")
	return reservedTopLevelKeys[first]
}

// RemoveProfile removes the profile whose name matches the provided
// profileName from the config file.
//
// It returns ErrProfileNotFound when there is nothing to remove, so a caller
// acting on a name a user typed can tell the difference between a removal and a
// no-op. Callers that treat removal as best effort can ignore it.
func (c *Config) RemoveProfile(profileName string) error {
	if profileName == "" {
		return errorcategory.Errorf(errorcategory.UserInput, "profile name cannot be empty")
	}

	if isReservedConfigKey(profileName) {
		return errorcategory.Errorf(errorcategory.UserInput, "%q is a reserved config key, not a profile", profileName)
	}

	runtimeViper := viper.GetViper()
	var err error
	var matched bool

	for _, entry := range listProfileEntries(runtimeViper) {
		if !entry.matches(profileName) {
			continue
		}

		matched = true

		runtimeViper, err = removeKey(runtimeViper, entry.key)
		if err != nil {
			return err
		}

		deleteLivemodeKey(LiveModeAPIKeyName, entry.name)
	}

	// A top-level profile is only found above when isProfile recognizes it, which
	// requires a display_name, and a profile whose name contains a period is
	// invisible to the enumeration entirely because viper reads that back as
	// nesting. Fall back to addressing the table by its full key path, which hits
	// exactly that profile and leaves siblings intact. The v2 path is tried first
	// for the same reason reads prefer it.
	if !matched {
		for _, key := range []string{ProfilesTableName + "." + profileName, profileName} {
			if !isProfileTable(runtimeViper, key) {
				continue
			}

			matched = true

			runtimeViper, err = removeKey(runtimeViper, key)
			if err != nil {
				return err
			}

			deleteLivemodeKey(LiveModeAPIKeyName, profileName)

			break
		}
	}

	if !matched {
		return ErrProfileNotFound
	}

	return writeConfig(runtimeViper)
}

// isProfileTable reports whether profileName addresses a table in the config
// file. It deliberately does not require a display_name: a profile written
// before display_name was always set, or one whose write was interrupted, still
// needs to be removable.
//
// Get is used rather than AllSettings because Get returns the raw, unflattened
// sub-map, so a name containing a period resolves to the table the user wrote
// rather than being split into a path.
func isProfileTable(v *viper.Viper, profileName string) bool {
	if !v.IsSet(profileName) {
		return false
	}

	switch v.Get(profileName).(type) {
	case map[string]interface{}, map[string]string:
		return true
	}

	return false
}

// RemoveAllProfiles removes all the profiles from the config file.
func (c *Config) RemoveAllProfiles() error {
	runtimeViper := viper.GetViper()
	var err error

	for _, entry := range listProfileEntries(runtimeViper) {
		runtimeViper, err = removeKey(runtimeViper, entry.key)
		if err != nil {
			return err
		}

		deleteLivemodeKey(LiveModeAPIKeyName, entry.name)
	}

	return writeConfig(runtimeViper)
}

// RemoveAuthFields removes only auth-related fields for the named profile,
// preserving non-auth settings like color.
func (c *Config) RemoveAuthFields(profileName string) error {
	runtimeViper := viper.GetViper()
	var matched bool

	for _, entry := range listProfileEntries(runtimeViper) {
		if !entry.matches(profileName) {
			continue
		}

		matched = true

		p := &Profile{ProfileName: entry.name}
		runtimeViper = p.deleteAuthFields(runtimeViper)
		deleteLivemodeKey(LiveModeAPIKeyName, entry.name)
	}

	// Profiles with a period in the name are nested tables that the enumeration
	// above cannot see. Clearing them by name is the only way for a user to log
	// out of one, so handle that case explicitly. deleteAuthFields clears both
	// layouts, so this covers a dotted profile wherever it lives.
	if !matched && isNestedProfileName(profileName) &&
		(runtimeViper.IsSet(ProfilesTableName+"."+profileName) || runtimeViper.IsSet(profileName)) {
		p := &Profile{ProfileName: profileName}
		runtimeViper = p.deleteAuthFields(runtimeViper)
		deleteLivemodeKey(LiveModeAPIKeyName, profileName)
	}

	deleteTopLevelLivemodeKey(UATKeychainItemKey)
	deleteTopLevelLivemodeKey(OAuthRefreshTokenKeychainKey)
	deleteTopLevelLivemodeKey(OAuthActiveContextKeychainKey)
	deleteTopLevelLivemodeKey(OAuthUATExpiresAtKeychainKey)

	// TODO: remove with legacy RAK/OIDC flow.
	if runtimeViper.IsSet(UserInfoName) {
		runtimeViper, _ = removeKey(runtimeViper, UserInfoName)
	}

	return writeConfig(runtimeViper)
}

// RemoveAllAuthFields removes only auth-related fields from all profiles,
// preserving non-auth settings like color.
func (c *Config) RemoveAllAuthFields() error {
	runtimeViper := viper.GetViper()

	for _, entry := range listProfileEntries(runtimeViper) {
		p := &Profile{ProfileName: entry.name}
		runtimeViper = p.deleteAuthFields(runtimeViper)
		deleteLivemodeKey(LiveModeAPIKeyName, entry.name)
	}

	deleteTopLevelLivemodeKey(UATKeychainItemKey)
	deleteTopLevelLivemodeKey(OAuthRefreshTokenKeychainKey)
	deleteTopLevelLivemodeKey(OAuthActiveContextKeychainKey)
	deleteTopLevelLivemodeKey(OAuthUATExpiresAtKeychainKey)

	// TODO: remove with legacy RAK/OIDC flow.
	if runtimeViper.IsSet(UserInfoName) {
		runtimeViper, _ = removeKey(runtimeViper, UserInfoName)
	}

	return writeConfig(runtimeViper)
}

func deleteLivemodeKey(key string, profile string) error {
	fieldID := profile + "." + key
	err := KeyRing.Remove(fieldID)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

func deleteTopLevelLivemodeKey(key string) error {
	err := KeyRing.Remove(key)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

// isNestedProfileName reports whether a profile name would be read back from
// the config file as a nested table rather than a single top-level one. Such
// profiles are skipped by every loop over AllSettings(), because viper joins and
// splits keys on "." and so cannot distinguish ["a.b"] from [a] -> [b].
func isNestedProfileName(profileName string) bool {
	return profileName != "" && strings.Contains(profileName, ".")
}

// isProfile identifies whether a top-level config entry pertains to a user
// profile. At the top level a profile is indistinguishable from a settings table
// except by its contents, so it is identified by its display_name field.
//
// Entries inside the v2 profiles table need no such guess: every table in there
// is a profile by definition. Use listProfileEntries to enumerate profiles in a
// way that covers both layouts.
func isProfile(value interface{}) bool {
	switch v := value.(type) {
	case map[string]interface{}:
		_, ok := v["display_name"]
		return ok
	case map[string]string:
		_, ok := v["display_name"]
		return ok
	}
	return false
}

// configVersion reports the schema version recorded in the config file. A v1 file
// records no config_version key, so an absent key reports ConfigVersionV1.
//
// It errors when the recorded version is one this binary cannot act on: either
// unreadable as a version number, or newer than MaxSupportedConfigVersion. Both
// are errors rather than a fallback to v1, because v1 means "flat layout" to every
// caller, and a flat write into a file whose profiles are nested lands a second
// copy that the read path then shadows — a write that reports success and is
// invisible to the next read.
//
// On success the version is always >= ConfigVersionV1; 0 is returned only with an
// error.
func configVersion(v *viper.Viper) (int, error) {
	raw := v.Get(ConfigVersionName)
	if raw == nil {
		return ConfigVersionV1, nil
	}

	// GetInt discards the cast error, so an unreadable value arrives here as 0 and
	// is caught by the range check rather than by inspecting the error.
	version := v.GetInt(ConfigVersionName)
	if version < ConfigVersionV1 {
		return 0, errorcategory.Errorf(errorcategory.Filesystem,
			"%s is set to %v, which is not a version number", ConfigVersionName, raw)
	}

	if version > MaxSupportedConfigVersion {
		return version, unsupportedConfigVersionError(version)
	}

	return version, nil
}

// isMigrated reports whether the config file stores profiles in the v2 layout,
// under the reserved profiles table.
func isMigrated(v *viper.Viper) bool {
	version, err := configVersion(v)
	return err == nil && version >= ConfigVersionV2
}

// profileTableKeyForWrite returns the key a whole profile table should be
// written to, matching the layout the config file already uses.
func profileTableKeyForWrite(v *viper.Viper, profileName string) string {
	if isMigrated(v) {
		return ProfilesTableName + "." + profileName
	}

	return profileName
}

// profileEntry is one profile found in the config file, in either layout.
type profileEntry struct {
	// name is the profile's name, e.g. "default". Keyring item IDs are prefixed
	// with this name in both layouts, so it is what credential lookups need.
	name string

	// key is the viper key of the profile's table: "default" in a v1 file,
	// "profiles.default" in a v2 file.
	key string

	// settings is the contents of the profile's table.
	settings map[string]interface{}
}

// matches reports whether this entry is the profile the user named, either by
// its table name or by its recorded profile_name.
func (e profileEntry) matches(profileName string) bool {
	if e.name == profileName {
		return true
	}

	recorded, _ := e.settings["profile_name"].(string)

	return recorded == profileName
}

// listProfileEntries returns every profile in the config file, reading both the
// v2 profiles table and the v1 top level.
//
// Both layouts are scanned because a migrated file can still acquire a top-level
// profile table: this CLI, and any plugin old enough not to know about the
// profiles table, writes one. A v1 entry is skipped when a v2 profile of the
// same name exists, so the migrated copy always wins, matching the read
// precedence in ReadProfileString.
func listProfileEntries(v *viper.Viper) []profileEntry {
	settings := v.AllSettings()
	entries := make([]profileEntry, 0, len(settings))
	nestedNames := make(map[string]bool)

	if nested, ok := toStringMap(settings[ProfilesTableName]); ok {
		for name, value := range nested {
			table, ok := toStringMap(value)
			if !ok || !holdsProfileFields(table) {
				continue
			}

			nestedNames[name] = true
			entries = append(entries, profileEntry{
				name:     name,
				key:      ProfilesTableName + "." + name,
				settings: table,
			})
		}
	}

	for name, value := range settings {
		if name == ProfilesTableName || nestedNames[name] || !isProfile(value) {
			continue
		}

		table, _ := toStringMap(value)
		entries = append(entries, profileEntry{name: name, key: name, settings: table})
	}

	// AllSettings returns a map, so sort for stable output and stable ordering of
	// the writes that follow.
	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	return entries
}

// holdsProfileFields reports whether a table inside the profiles table is a
// profile rather than a container of them.
//
// Viper splits keys on ".", so a hand-written profile whose name contains a
// period reads back as nesting: profiles."a.b" becomes profiles -> a -> b. Every
// profile field is a scalar, so requiring at least one keeps such a container
// from being listed and removed as if it were a profile named "a". Migration
// never produces one — it leaves a dotted profile at the top level, where the
// existing by-name fallbacks handle it.
func holdsProfileFields(table map[string]interface{}) bool {
	for _, value := range table {
		if _, isTable := toStringMap(value); !isTable {
			return true
		}
	}

	return false
}

// toStringMap normalizes the two map shapes viper hands back for a TOML table.
func toStringMap(value interface{}) (map[string]interface{}, bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		return v, true
	case map[string]string:
		normalized := make(map[string]interface{}, len(v))
		for key, item := range v {
			normalized[key] = item
		}

		return normalized, true
	}

	return nil, false
}

// WriteConfigField updates a configuration field and writes the updated
// configuration to disk.
func (c *Config) WriteConfigField(field string, value interface{}) error {
	runtimeViper := viper.GetViper()
	runtimeViper.Set(field, value)

	return writeConfig(runtimeViper)
}

// writeConfig writes a viper instance to the config file and syncs the global viper.
func writeConfig(runtimeViper *viper.Viper) error {
	// Refuse to write a file whose layout version this binary does not know. Both
	// candidate layouts are a guess at that point, and the flat guess is silently
	// shadowed by the nested copy, so a login would report success while leaving
	// the credential unreadable. Checked on the global viper, not runtimeViper:
	// writeProfile passes a scratch viper that has not merged the file yet, which
	// is the same reason configFieldForWrite reads the global.
	if _, err := configVersion(viper.GetViper()); err != nil {
		return err
	}

	profilesFile := viper.ConfigFileUsed()
	runtimeViper.SetConfigFile(profilesFile)
	configType := strings.TrimPrefix(filepath.Ext(profilesFile), ".")
	runtimeViper.SetConfigType(configType)

	if err := fsutil.RefuseWriteThroughSymlinkOS(profilesFile, filepath.Dir(filepath.Dir(profilesFile)), filepath.Base(profilesFile)); err != nil {
		return err
	}

	if err := runtimeViper.WriteConfig(); err != nil {
		return err
	}

	return ReloadConfigFile()
}

// ReloadConfigFile re-reads the config file into the global viper, replacing what
// is already loaded rather than merging into it. Anything that rewrites the file
// behind viper's back — the config migration, for one — needs this to see its own
// changes.
//
// The reset matters: ReadInConfig merges with existing values, so keys the
// rewrite removed would otherwise linger in memory.
func ReloadConfigFile() error {
	profilesFile := viper.ConfigFileUsed()
	configType := strings.TrimPrefix(filepath.Ext(profilesFile), ".")

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

	if err := fsutil.RefuseWriteThroughSymlinkOS(dir, filepath.Dir(dir), "config directory"); err != nil {
		return err
	}

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
