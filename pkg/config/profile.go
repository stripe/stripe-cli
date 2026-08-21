package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// Compartment represents a Stripe compartment from the OIDC userinfo response.
type Compartment struct {
	CompartmentID string `json:"compartment_id" mapstructure:"compartment_id" toml:"compartment_id"`
	Livemode      bool   `json:"livemode"        mapstructure:"livemode"        toml:"livemode"`
}

// UserInfo mirrors the OIDC userinfo endpoint response and is persisted as a
// nested table in the profile config.
type UserInfo struct {
	Compartments []Compartment `json:"https://stripe.com/compartments" mapstructure:"compartments" toml:"compartments"`
}

// Profile handles all things related to managing the project specific configurations
type Profile struct {
	DeviceName             string
	ProfileName            string
	APIKey                 string
	LiveModeAPIKey         string
	LiveModePublishableKey string
	TestModeAPIKey         string
	TestModePublishableKey string
	DisplayName            string
	AccountID              string
	UserID                 string
	SandboxClaimURL        string
	SandboxExpiresAt       string
	UAT                    string
	UserInfo               *UserInfo
}

// config key names
const (
	AccountIDName              = "account_id"
	UserIDName                 = "user_id"
	DeviceNameName             = "device_name"
	DisplayNameName            = "display_name"
	IsTermsAcceptanceValidName = "is_terms_acceptance_valid"
	TestModeAPIKeyName         = "test_mode_api_key"
	TestModePubKeyName         = "test_mode_pub_key"
	TestModeKeyExpiresAtName   = "test_mode_key_expires_at"
	LiveModeAPIKeyName         = "live_mode_api_key"
	LiveModePubKeyName         = "live_mode_pub_key"
	LiveModeKeyExpiresAtName   = "live_mode_key_expires_at"
	SandboxClaimURLName        = "sandbox_claim_url"
	SandboxExpiresAtName       = "sandbox_expires_at"
	UserInfoName               = "user_info"

	// ConfigVersionName is the top-level key recording the config file's schema
	// version. It is absent from v1 files, which is how an unmigrated file is
	// recognized.
	ConfigVersionName = "config_version"

	// ProfilesTableName is the reserved top-level table that holds every profile
	// in a v2 config file.
	ProfilesTableName = "profiles"
)

// ConfigVersionV1 is the original layout, with each profile as a top-level table.
// A v1 file records no config_version key at all; the constant exists so that
// version comparisons don't have to spell out that absence.
const ConfigVersionV1 = 1

// ConfigVersionV2 is the schema version in which every profile moved under the
// reserved ProfilesTableName table, so that profile names can no longer collide
// with top-level CLI settings.
const ConfigVersionV2 = 2

// MaxSupportedConfigVersion is the newest config.toml layout this binary can act
// on. A file recording a higher version was written by a newer CLI, whose layout
// this build has no way to know.
const MaxSupportedConfigVersion = ConfigVersionV2

const UATKeychainItemKey = "uat"

const (
	// DateStringFormat is the format for expiredAt date
	DateStringFormat = "2006-01-02"

	// KeyValidInDays is the number of days the API key is valid for
	KeyValidInDays = 90

	// KeyManagementService is the key management service name
	KeyManagementService = "StripeCLI"
)

// KeyRing is the global secure credential store.
var KeyRing keyring.SecureStore

// authFieldNames are the config fields that are removed on login/logout.
// Non-auth fields like "color" are preserved.
var authFieldNames = []string{
	DeviceNameName,
	DisplayNameName,
	AccountIDName,
	UserIDName,
	IsTermsAcceptanceValidName,
	TestModeAPIKeyName,
	TestModePubKeyName,
	TestModeKeyExpiresAtName,
	LiveModeAPIKeyName,
	LiveModePubKeyName,
	LiveModeKeyExpiresAtName,
	"profile_name",
	// sandbox-specific fields from stripe sandbox create
	SandboxClaimURLName,
	SandboxExpiresAtName,
	// legacy field names from older config formats
	"secret_key",
	"api_key",
	"publishable_key",
	"test_mode_publishable_key",
	// experimental settings are auth-session-scoped
	"experimental",
}

// ValidateProfileName reports whether name can be used as a profile name.
//
// Profile fields are addressed as <profile>.<field> keys in viper, which uses
// "." as its path separator. A profile name containing a period is therefore
// indistinguishable from a nested table: viper reads ["a.b"] back as a -> b, so
// the profile becomes invisible to every operation that enumerates top-level
// tables (listing, removal, clearing credentials) even though reads still work.
// An empty name produces a leading-period key that viper silently drops, so
// writes to it are discarded without error.
//
// Every other character round-trips correctly and is allowed.
//
// TODO: return errorcategory.UserInputErrorf once master is merged into v2 and
// the errorcategory package is available on this branch.
func ValidateProfileName(name string) error {
	if name == "" {
		return fmt.Errorf("profile name cannot be empty")
	}

	if strings.Contains(name, ".") {
		return fmt.Errorf("profile name %q cannot contain a period; use a hyphen or underscore instead", name)
	}

	return nil
}

// profileExists reports whether the config file already contains a table for
// this profile. This checks for the table itself rather than any particular
// field, so that a partially written profile (one with no display_name, which a
// profile abandoned part-way through login can be) is still recognized.
func (p *Profile) profileExists() bool {
	return viper.IsSet(p.nestedProfileTable()) || viper.IsSet(p.ProfileName)
}

// ValidateProfileNameForWrite validates the profile name before writing to it.
//
// Profiles that already exist are allowed through unchanged: reads against them
// work correctly, and refusing to write would leave users unable to update a
// profile they are actively using. Only the creation of new invalid names is
// blocked.
//
// This grandfather clause can be dropped once the config migration renames
// existing dotted profiles, at which point the rule becomes unconditional.
func (p *Profile) ValidateProfileNameForWrite() error {
	if p.profileExists() {
		return nil
	}

	return ValidateProfileName(p.ProfileName)
}

// CreateProfile creates a profile when logging in
func (p *Profile) CreateProfile() error {
	// Validate up front so a rejected name leaves both the config file and the
	// keyring untouched: the calls below delete existing credentials before
	// anything is written back. writeProfile revalidates as the backstop for the
	// write paths that do not go through here.
	if err := p.ValidateProfileNameForWrite(); err != nil {
		return err
	}

	// Remove only auth-related keys under existing profile first
	v := p.deleteAuthFields(viper.GetViper())

	// Fail open to avoid blocking login
	p.deleteLivemodeValue(LiveModeAPIKeyName)

	// user_info is top-level; remove it before re-writing so stale data is never kept
	if v.IsSet(UserInfoName) {
		var err error
		v, err = removeKey(v, UserInfoName)
		if err != nil {
			return err
		}
	}

	writeErr := p.writeProfile(v)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (p *Profile) deleteAuthFields(v *viper.Viper) *viper.Viper {
	for _, field := range authFieldNames {
		v = p.safeRemove(v, field)
	}
	return v
}

// GetColor gets the color setting for the user based on the flag or the
// persisted color stored in the config file
func (p *Profile) GetColor() (string, error) {
	color := viper.GetString(ColorName)
	if color != "" {
		return color, nil
	}

	color = p.ReadProfileString(ColorName)
	switch color {
	case "", ColorAuto:
		return ColorAuto, nil
	case ColorOn:
		return ColorOn, nil
	case ColorOff:
		return ColorOff, nil
	default:
		return "", fmt.Errorf("color value not supported: %s", color)
	}
}

// GetDeviceName returns the configured device name
func (p *Profile) GetDeviceName() (string, error) {
	if os.Getenv("STRIPE_DEVICE_NAME") != "" {
		return os.Getenv("STRIPE_DEVICE_NAME"), nil
	}

	if p.DeviceName != "" {
		return p.DeviceName, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return p.ReadProfileString(DeviceNameName), nil
	}

	return "", validators.ErrDeviceNameNotConfigured
}

// GetAccountID returns the accountId for the given profile.
func (p *Profile) GetAccountID() (string, error) {
	if p.AccountID != "" {
		return p.AccountID, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return p.ReadProfileString(AccountIDName), nil
	}

	return "", validators.ErrAccountIDNotConfigured
}

// GetUserID returns the user ID for the given profile.
func (p *Profile) GetUserID() (string, error) {
	if p.UserID != "" {
		return p.UserID, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return p.ReadProfileString(UserIDName), nil
	}

	return "", nil
}

// HasOverrideAPIKey reports whether an in-memory API key override is active
// (via STRIPE_API_KEY env var or --api-key flag).
func (p *Profile) HasOverrideAPIKey() bool {
	return os.Getenv("STRIPE_API_KEY") != "" || p.APIKey != ""
}

// isLivemodeKey reports whether a key's prefix indicates live mode.
func isLivemodeKey(key string) bool {
	parts := strings.SplitN(key, "_", 3)
	return len(parts) >= 2 && parts[1] == "live"
}

// HasAPIKey reports whether an API key is available for the given mode without
// reading its value. For live mode this checks the keyring key list only,
// avoiding OS-level auth prompts (e.g. macOS Keychain) that would be required
// to access the secret data.
func (p *Profile) HasAPIKey(livemode bool) bool {
	if key := os.Getenv("STRIPE_API_KEY"); key != "" {
		return isLivemodeKey(key) == livemode
	}
	if p.APIKey != "" {
		return isLivemodeKey(p.APIKey) == livemode
	}

	if !livemode {
		if err := viper.ReadInConfig(); err != nil {
			return false
		}
		if p.profileFieldIsSet("secret_key") {
			p.RegisterAlias(TestModeAPIKeyName, "secret_key")
		} else if p.profileFieldIsSet("api_key") {
			p.RegisterAlias(TestModeAPIKeyName, "api_key")
		}
		return p.ReadProfileString(TestModeAPIKeyName) != ""
	}

	if KeyRing == nil {
		return false
	}
	_, err := KeyRing.Get(p.GetConfigField(LiveModeAPIKeyName))
	return err == nil
}

// GetAPIKey will return the existing key for the given profile
func (p *Profile) GetAPIKey(livemode bool) (string, error) {
	envKey := os.Getenv("STRIPE_API_KEY")
	if envKey != "" {
		err := validators.APIKey(envKey)
		if err != nil {
			return "", err
		}

		return envKey, nil
	}

	if p.APIKey != "" {
		err := validators.APIKey(p.APIKey)
		if err != nil {
			return "", err
		}

		return p.APIKey, nil
	}

	var key string
	var err error

	// Try to fetch the API key from the configuration file
	if !livemode {
		// If the user doesn't have an api_key field set, they might be using an
		// old configuration so try to read from secret_key
		if p.profileFieldIsSet("secret_key") {
			p.RegisterAlias(TestModeAPIKeyName, "secret_key")
		} else if p.profileFieldIsSet("api_key") {
			p.RegisterAlias(TestModeAPIKeyName, "api_key")
		}

		if err := viper.ReadInConfig(); err == nil {
			key = p.ReadProfileString(TestModeAPIKeyName)
		}
	} else {
		p.redactAllLivemodeValues()
		key, err = p.retrieveLivemodeValue(LiveModeAPIKeyName)
		if err != nil {
			return "", errors.New("your live mode API key needs to be re-configured. Run `stripe login` to re-authenticate")
		}
	}

	if key != "" {
		err = validators.APIKey(key)
		if err != nil {
			return "", err
		}
		return key, nil
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// GetExpiresAt returns the API key expirary date
func (p *Profile) GetExpiresAt(livemode bool) (time.Time, error) {
	var timeString string

	if livemode {
		timeString = p.ReadProfileString(LiveModeKeyExpiresAtName)
	} else {
		timeString = p.ReadProfileString(TestModeKeyExpiresAtName)
	}

	if timeString != "" {
		expiresAt, err := time.Parse(DateStringFormat, timeString)
		if err != nil {
			return time.Time{}, err
		}
		return expiresAt, nil
	}

	return time.Time{}, validators.ErrAPIKeyNotConfigured
}

// GetPublishableKey returns the publishable key for the user
func (p *Profile) GetPublishableKey(livemode bool) (string, error) {
	var fieldID string
	var key string

	if livemode {
		fieldID = LiveModePubKeyName
	} else {
		fieldID = TestModePubKeyName

		if p.profileFieldIsSet("publishable_key") {
			p.RegisterAlias(TestModePubKeyName, "publishable_key")
		}
		// there is a bug with viper.GetStringMapString when the key name is too long, which makes
		// `config --list --project-name <project_name>` unable to read the project specific config
		if p.profileFieldIsSet("test_mode_publishable_key") {
			p.RegisterAlias(TestModePubKeyName, "test_mode_publishable_key")
		}
	}

	err := viper.ReadInConfig()
	if err != nil {
		return "", err
	}

	key = p.ReadProfileString(fieldID)
	if key != "" {
		return key, nil
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// GetDisplayName returns the account display name of the user
func (p *Profile) GetDisplayName() string {
	if err := viper.ReadInConfig(); err == nil {
		return p.ReadProfileString(DisplayNameName)
	}

	return ""
}

// GetConfigField returns "<profile>.<field>", e.g. "default.account_id". This is
// both the v1 flat config path and, for livemode secrets, the keyring item ID.
//
// The keyring IDs are already on users' machines, so this string is frozen: it
// cannot gain the v2 "profiles." prefix without orphaning every stored
// credential. For config access use ReadProfileString, profileFieldIsSet, or
// configFieldForWrite, which understand both layouts.
func (p *Profile) GetConfigField(field string) string {
	return p.ProfileName + "." + field
}

// nestedProfileTable returns the v2 key of this profile's table, e.g.
// "profiles.default".
func (p *Profile) nestedProfileTable() string {
	return ProfilesTableName + "." + p.ProfileName
}

// nestedConfigField returns the v2 configuration path for a profile field,
// e.g. "profiles.default.account_id".
func (p *Profile) nestedConfigField(field string) string {
	return p.nestedProfileTable() + "." + field
}

// ReadProfileString reads a profile field from the config file, preferring the
// v2 nested layout and falling back to the v1 flat layout.
//
// Both layouts are supported indefinitely. A config file that has never been
// migrated stays fully readable.
func (p *Profile) ReadProfileString(field string) string {
	if value := viper.GetString(p.nestedConfigField(field)); value != "" {
		return value
	}

	return viper.GetString(p.GetConfigField(field))
}

// profileFieldIsSet reports whether a profile field is present in either layout.
func (p *Profile) profileFieldIsSet(field string) bool {
	return viper.IsSet(p.nestedConfigField(field)) || viper.IsSet(p.GetConfigField(field))
}

// configFieldForWrite returns the path a profile field should be written to,
// matching the layout the config file already uses.
//
// Reads tolerate either layout, but a write has to pick one, so writes follow
// config_version. An unmigrated file has no config_version key, so GetInt
// returns 0 and writes stay flat.
func (p *Profile) configFieldForWrite(field string) string {
	// The layout is read from the global viper, which is the instance that has
	// the config file loaded. writeProfile is handed a scratch viper that has not
	// merged the file in yet at the point the fields are set.
	if isMigrated(viper.GetViper()) {
		return p.nestedConfigField(field)
	}

	return p.GetConfigField(field)
}

// RegisterAlias registers an alias for a given key in both layouts, so that
// legacy field names (secret_key, api_key, publishable_key) keep resolving in a
// migrated config file as well as an unmigrated one.
func (p *Profile) RegisterAlias(alias, key string) {
	viper.RegisterAlias(p.GetConfigField(alias), p.GetConfigField(key))
	viper.RegisterAlias(p.nestedConfigField(alias), p.nestedConfigField(key))
}

// WriteConfigField updates a configuration field and writes the updated
// configuration to disk.
func (p *Profile) WriteConfigField(field, value string) error {
	viper.ReadInConfig()

	if err := p.ValidateProfileNameForWrite(); err != nil {
		return err
	}

	viper.Set(p.configFieldForWrite(field), value)
	return writeConfig(viper.GetViper())
}

// DeleteConfigField deletes a configuration field.
func (p *Profile) DeleteConfigField(field string) error {
	v, err := p.deleteProfileField(viper.GetViper(), field)
	if err != nil {
		return err
	}

	// delete livemode redacted values from config and full values from keyring
	if field == LiveModeAPIKeyName {
		p.deleteLivemodeValue(field)
	}

	return p.writeProfile(v)
}

func (p *Profile) writeProfile(runtimeViper *viper.Viper) error {
	profilesFile := viper.ConfigFileUsed()

	// Validate before any mutation so a rejected name leaves the config file and
	// the keyring untouched. This is the single funnel every profile write goes
	// through, so commands cannot reintroduce an invalid name by bypassing the
	// earlier checks in `stripe login` and `stripe sandbox create`.
	if err := p.ValidateProfileNameForWrite(); err != nil {
		return err
	}

	err := makePath(profilesFile)
	if err != nil {
		return err
	}

	if p.DeviceName != "" {
		runtimeViper.Set(p.configFieldForWrite(DeviceNameName), strings.TrimSpace(p.DeviceName))
	}

	if p.LiveModeAPIKey != "" {
		expiresAt := getKeyExpiresAt()
		runtimeViper.Set(p.configFieldForWrite(LiveModeKeyExpiresAtName), expiresAt)

		// // store redacted key in config
		runtimeViper.Set(p.configFieldForWrite(LiveModeAPIKeyName), RedactAPIKey(strings.TrimSpace(p.LiveModeAPIKey)))

		// // store actual key in secure keyring
		if err := p.saveLivemodeValue(LiveModeAPIKeyName, strings.TrimSpace(p.LiveModeAPIKey), "Live mode API key"); err != nil {
			return err
		}
	}

	if p.LiveModePublishableKey != "" {
		runtimeViper.Set(p.configFieldForWrite(LiveModePubKeyName), strings.TrimSpace(p.LiveModePublishableKey))
	}

	if p.TestModeAPIKey != "" {
		runtimeViper.Set(p.configFieldForWrite(TestModeAPIKeyName), strings.TrimSpace(p.TestModeAPIKey))
		runtimeViper.Set(p.configFieldForWrite(TestModeKeyExpiresAtName), getKeyExpiresAt())
	}

	if p.TestModePublishableKey != "" {
		runtimeViper.Set(p.configFieldForWrite(TestModePubKeyName), strings.TrimSpace(p.TestModePublishableKey))
	}

	if p.DisplayName != "" {
		runtimeViper.Set(p.configFieldForWrite(DisplayNameName), strings.TrimSpace(p.DisplayName))
	}

	if p.AccountID != "" {
		runtimeViper.Set(p.configFieldForWrite(AccountIDName), strings.TrimSpace(p.AccountID))
	}

	if p.UserID != "" {
		runtimeViper.Set(p.configFieldForWrite(UserIDName), strings.TrimSpace(p.UserID))
	}

	if p.SandboxClaimURL != "" {
		runtimeViper.Set(p.configFieldForWrite(SandboxClaimURLName), strings.TrimSpace(p.SandboxClaimURL))
	}

	if p.SandboxExpiresAt != "" {
		runtimeViper.Set(p.configFieldForWrite(SandboxExpiresAtName), strings.TrimSpace(p.SandboxExpiresAt))
	}

	if KeyRing != nil {
		if p.UAT != "" {
			if err := KeyRing.Set(UATKeychainItemKey, []byte(strings.TrimSpace(p.UAT)), "Stripe CLI user access token"); err != nil {
				return err
			}
		} else {
			if err := KeyRing.Remove(UATKeychainItemKey); err != nil && !errors.Is(err, keyring.ErrKeyNotFound) {
				return err
			}
		}
	}

	if p.UserInfo != nil {
		runtimeViper.Set(UserInfoName, p.UserInfo)
	}

	runtimeViper.MergeInConfig()

	// Do this after we merge the old configs in
	if p.TestModeAPIKey != "" {
		runtimeViper = p.safeRemove(runtimeViper, "secret_key")
		runtimeViper = p.safeRemove(runtimeViper, "api_key")
	}

	if p.TestModePublishableKey != "" {
		runtimeViper = p.safeRemove(runtimeViper, "publishable_key")
	}

	return writeConfig(runtimeViper)
}

// deleteProfileField removes a profile field from both the v2 nested layout and
// the v1 flat layout. An older CLI or plugin can write a flat copy into a
// migrated file, and clearing only one layout is not a partial delete but a
// no-op: whichever copy is left is the one the read path returns.
func (p *Profile) deleteProfileField(v *viper.Viper, field string) (*viper.Viper, error) {
	for _, path := range []string{p.nestedConfigField(field), p.GetConfigField(field)} {
		if !v.IsSet(path) {
			continue
		}

		newViper, err := removeKey(v, path)
		if err != nil {
			return nil, err
		}

		v = newViper
	}

	return v, nil
}

func (p *Profile) safeRemove(v *viper.Viper, key string) *viper.Viper {
	newViper, err := p.deleteProfileField(v, key)
	if err != nil {
		// I don't want to fail the entire login process on not being able to remove
		// the old secret_key field, so keep the viper we already have.
		return v
	}

	return newViper
}

// redactAllLivemodeValues redacts all livemode values in the local config file
func (p *Profile) redactAllLivemodeValues() {
	color := ansi.Color(os.Stdout)

	if err := viper.ReadInConfig(); err == nil {
		// if the config file has expires at date, then it is using the new livemode key storage
		if p.profileFieldIsSet(LiveModeAPIKeyName) {
			key := p.ReadProfileString(LiveModeAPIKeyName)
			if key == "" || len(key) < 12 {
				p.DeleteConfigField(LiveModeAPIKeyName)
				return
			}

			if !isRedactedAPIKey(key) {
				fmt.Println(color.Yellow(`
(!) Livemode value found for the field '` + LiveModeAPIKeyName + `' in your config file.
Livemode values from the config file will be redacted and will not be used.`))

				p.WriteConfigField(LiveModeAPIKeyName, RedactAPIKey(key))
			}
		}
	}
}

// RedactAPIKey returns a redacted version of API keys. The first 8 and last 4
// characters are not redacted, everything else is replaced by "*" characters.
//
// It panics if the provided string has less than 12 characters.
func RedactAPIKey(apiKey string) string {
	var b strings.Builder

	b.WriteString(apiKey[0:8])                         // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(strings.Repeat("*", len(apiKey)-12)) // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)
	b.WriteString(apiKey[len(apiKey)-4:])              // #nosec G104 (gosec bug: https://github.com/securego/gosec/issues/267)

	return b.String()
}

// isRedactedAPIKey checks if the input string is a refacted api key
func isRedactedAPIKey(apiKey string) bool {
	keyParts := strings.Split(apiKey, "_")
	if len(keyParts) < 3 {
		return false
	}

	if keyParts[0] != "sk" && keyParts[0] != "rk" {
		return false
	}

	if RedactAPIKey(apiKey) != apiKey {
		return false
	}

	return true
}

func getKeyExpiresAt() string {
	return time.Now().AddDate(0, 0, KeyValidInDays).UTC().Format(DateStringFormat)
}

// saveLivemodeValue saves livemode value of given key in keyring
func (p *Profile) saveLivemodeValue(field, value, description string) error {
	fieldID := p.GetConfigField(field)
	return KeyRing.Set(fieldID, []byte(value), description)
}

// retrieveLivemodeValue retrieves livemode value of given key in keyring
func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
	fieldID := p.GetConfigField(key)
	data, err := KeyRing.Get(fieldID)
	if err != nil {
		return "", validators.ErrAPIKeyNotConfigured
	}
	return string(data), nil
}

// deleteLivemodeValue deletes livemode value of given key in keyring
func (p *Profile) deleteLivemodeValue(key string) error {
	fieldID := p.GetConfigField(key)
	err := KeyRing.Remove(fieldID)
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

// GetUserInfo reads the stored UserInfo from the profile config.
// Returns nil, nil when no user_info has been saved yet.
func (p *Profile) GetUserInfo() (*UserInfo, error) {
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	if !viper.IsSet(UserInfoName) {
		return nil, nil
	}

	var ui UserInfo
	if err := viper.UnmarshalKey(UserInfoName, &ui); err != nil {
		return nil, err
	}

	return &ui, nil
}

// SessionCredentials are the credentials needed for this session
type SessionCredentials struct {
	UAT        string `json:"uat"`
	PrivateKey string `json:"private_key"`
	AccountID  string `json:"account_id"`
}

// GetSessionCredentials retrieves the session credentials from the keyring
func (p *Profile) GetSessionCredentials() (*SessionCredentials, error) {
	key := p.GetConfigField("stripe_cli_session")
	data, err := KeyRing.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, errors.New("no session")
		}
		return nil, err
	}

	creds := SessionCredentials{}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, err
	}

	currentAccountID, err := p.GetAccountID()
	if err != nil {
		return nil, err
	}

	if creds.AccountID == "" || creds.AccountID != currentAccountID {
		return nil, errors.New("found a session, but it doesn't match your current account")
	}

	return &creds, nil
}
