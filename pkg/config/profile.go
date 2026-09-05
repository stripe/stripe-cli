package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
	"github.com/stripe/stripe-cli/pkg/stripe"
	"github.com/stripe/stripe-cli/pkg/validators"
)

var printActiveContextOnce sync.Once

// OAuthTokenRefresher is called by ResolveCredentials when an OAK token is
// expired or about to expire. It refreshes the token and updates p in-place.
// Set by the login package via init().
var OAuthTokenRefresher func(p *Profile) error

// refreshMu serializes concurrent refresh attempts so only one goroutine hits
// the token endpoint at a time; others re-use the result after the lock.
var refreshMu sync.Mutex

// AuthorizedAccount represents a Stripe account accessible to an OAuth token.
type AuthorizedAccount struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Modes []string `json:"modes"`
}

// Compartment represents a Stripe workspace from the OIDC userinfo response.
// TODO: remove with legacy RAK/OIDC flow.
type Compartment struct {
	CompartmentID string `json:"compartment_id" mapstructure:"compartment_id" toml:"compartment_id"`
	Livemode      bool   `json:"livemode"        mapstructure:"livemode"        toml:"livemode"`
}

// UserInfo mirrors the OIDC userinfo endpoint response and is persisted as a
// nested table in the profile config.
// TODO: remove with legacy RAK/OIDC flow.
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
	TerminalPOSDeviceID    string
	DisplayName            string
	AccountID              string
	UserID                 string
	SandboxClaimURL        string
	SandboxExpiresAt       string
	UAT                    string
	UserInfo               *UserInfo // TODO: remove with legacy RAK/OIDC flow

	// OAuthAccessBaseURL is the access-srv base URL to use for token refresh
	// and revocation. Set at startup from the --access-base flag; not persisted.
	OAuthAccessBaseURL string
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
	UserInfoName               = "user_info" // TODO: remove with legacy RAK/OIDC flow

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

func unsupportedConfigVersionError(version int) error {
	return errorcategory.Errorf(errorcategory.Filesystem,
		"%s is %d, but this Stripe CLI understands up to %d. Upgrade the CLI: https://docs.stripe.com/stripe-cli/upgrade",
		ConfigVersionName, version, MaxSupportedConfigVersion,
	)
}

const UATKeychainItemKey = "uat"

// OAuthActiveContextKeychainKey is the keyring key for the active OAuth context.
const OAuthActiveContextKeychainKey = "oauth_active_context"

// OAuthRefreshTokenKeychainKey is the keyring key for the OAuth refresh token.
const OAuthRefreshTokenKeychainKey = "oauth_refresh_token"

// OAuthUATExpiresAtKeychainKey is the keyring key for the UAT expiry time (RFC3339).
const OAuthUATExpiresAtKeychainKey = "oauth_uat_expires_at"

// ActiveContext identifies the account and mode that is currently selected.
type ActiveContext struct {
	AccountID string `json:"account_id"`
	Livemode  bool   `json:"livemode"`
}

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

// profileExists reports whether the config file already contains a table for
// this profile. This checks for the table itself rather than any particular
// field, so that a partially written profile (one with no display_name, which a
// profile abandoned part-way through login can be) is still recognized.
func (p *Profile) profileExists() bool {
	return viper.IsSet(p.nestedProfileTable()) || viper.IsSet(p.ProfileName)
}

// warnLegacyProfileNameOnce guards the deprecation warning so it prints at most
// once per process.
var warnLegacyProfileNameOnce sync.Once

// WarnIfLegacyProfileName tells the user when the active profile has a period in
// its name.
//
// Profile fields are addressed as <profile>.<field> keys in viper, which uses
// "." as its path separator. A profile name containing a period is therefore
// indistinguishable from a nested table: viper reads ["a.b"] back as a -> b, so
// the profile becomes invisible to every operation that enumerates top-level
// tables, even though reads against it keep working. That makes this warning the
// only signal the user gets that the profile cannot be listed by name.
//
// It only fires for a profile that is actually in the config file, so that a
// period in a name nobody is using stays silent.
func (p *Profile) WarnIfLegacyProfileName() {
	if !strings.Contains(p.ProfileName, ".") || !p.profileExists() {
		return
	}

	warnLegacyProfileNameOnce.Do(func() {
		color := ansi.Color(os.Stderr)
		fmt.Fprintln(os.Stderr, color.Yellow(fmt.Sprintf(`
(!) The profile %[1]q contains a period, which can cause unexpected
behavior and will stop being accepted in a future release. We strongly
recommend migrating it: log in under a new name and remove the old one.
  stripe login --project-name %[2]s
  stripe config --remove-profile %[1]s`,
			p.ProfileName,
			strings.ReplaceAll(p.ProfileName, ".", "-"))))
	})
}

// ValidateProfileName reports whether name can be used as a profile name.
//
// A period is rejected because it is viper's path separator, which hides the
// profile from every operation that enumerates top-level tables; see
// WarnIfLegacyProfileName for the full explanation. An empty name produces a
// leading-period key that viper silently drops, so writes to it are discarded
// without error.
//
// Every other character round-trips correctly and is allowed.
func ValidateProfileName(name string) error {
	if name == "" {
		return errorcategory.UserInputErrorf("profile name cannot be empty")
	}

	if strings.Contains(name, ".") {
		return errorcategory.UserInputErrorf("profile name %q cannot contain a period; use a hyphen or underscore instead", name)
	}

	return nil
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

	// TODO: remove with legacy RAK/OIDC flow.
	// user_info is top-level; remove it before re-writing so stale data is never kept.
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
		// safeRemove clears the field from whichever layouts it appears in, so a
		// file that a v2 CLI migrated (or that this CLI wrote a shadow copy into)
		// is fully cleared rather than half-cleared. Failure to remove a key
		// should not break the login flow.
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
		return "", errorcategory.Errorf(errorcategory.Filesystem, "color value not supported: %s", color)
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
			return "", fmt.Errorf(
				"invalid STRIPE_API_KEY environment variable: %w. STRIPE_API_KEY takes precedence over the CLI config file; unset it or pass --api-key with a valid key",
				err,
			)
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
			return "", errorcategory.New(errorcategory.Auth, "your live mode API key needs to be re-configured. Run `stripe login` to re-authenticate")
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

// GetTerminalPOSDeviceID returns the device id from the config for Terminal quickstart to use
func (p *Profile) GetTerminalPOSDeviceID() string {
	if err := viper.ReadInConfig(); err == nil {
		return p.ReadProfileString("terminal_pos_device_id")
	}

	return ""
}

// GetConfigField returns the flat (v1) configuration path for a profile field,
// e.g. "default.account_id".
//
// This same string is used verbatim as the keyring item ID for livemode
// secrets, so it must not gain the v2 "profiles." prefix: changing it would
// orphan every credential already stored in the OS keychain. To read a config
// value, use ReadProfileString or profileFieldIsSet, which understand both the
// v1 and v2 layouts. To write one, use configFieldForWrite.
func (p *Profile) GetConfigField(field string) string {
	return p.ProfileName + "." + field
}

// nestedProfileTable returns the v2 key of this profile's table, e.g.
// "profiles.default". In v2, profiles live under a reserved top-level "profiles"
// table so that profile names can no longer collide with CLI settings.
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
// migrated stays fully readable, and one that a v2 CLI migrated is readable from
// this release line without the user having to undo anything.
func (p *Profile) ReadProfileString(field string) string {
	if value := viper.GetString(p.nestedConfigField(field)); value != "" {
		return value
	}

	return viper.GetString(p.GetConfigField(field))
}

// ReadProfileStringMap reads a nested table inside a profile, such as the docs
// preferences, from whichever layout holds it.
func (p *Profile) ReadProfileStringMap(field string) map[string]string {
	if value := viper.GetStringMapString(p.nestedConfigField(field)); len(value) > 0 {
		return value
	}

	return viper.GetStringMapString(p.GetConfigField(field))
}

// profileFieldIsSet reports whether a profile field is present in either layout.
func (p *Profile) profileFieldIsSet(field string) bool {
	return viper.IsSet(p.nestedConfigField(field)) || viper.IsSet(p.GetConfigField(field))
}

// configFieldForWrite returns the path a profile field should be written to,
// matching the layout the config file already uses.
//
// Reads tolerate either layout, but a write has to pick one. Writing a flat
// field into a migrated file would create a second, shadow copy of the profile
// at the top level, which the read above would then shadow in turn — so the
// user's own write would be invisible to the CLI that just made it. An
// unmigrated file has no config_version key, so GetInt returns 0 and writes stay
// flat. Nothing here ever sets config_version: this release line follows the
// layout it finds and never migrates.
func (p *Profile) configFieldForWrite(field string) string {
	// The layout is read from the global viper, which is the instance that has
	// the config file loaded. writeProfile is handed a scratch viper that has not
	// merged the file in yet at the point the fields are set.
	if isMigrated(viper.GetViper()) {
		return p.nestedConfigField(field)
	}

	return p.GetConfigField(field)
}

// deleteProfileField removes a profile field from both the v2 nested layout and
// the v1 flat layout, so a delete cannot leave a stale copy of the field behind
// in a file that happens to contain both.
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

	// TODO: remove with legacy RAK/OIDC flow.
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

// SessionCredentials are the credentials needed for this session
type SessionCredentials struct {
	UAT        string `json:"uat"`
	PrivateKey string `json:"private_key"`
	AccountID  string `json:"account_id"`
}

// GetUAT retrieves the user access token from the keyring.
// Returns an empty string if no UAT is stored.
func (p *Profile) GetUAT() (string, error) {
	if KeyRing == nil {
		return "", nil
	}
	data, err := KeyRing.Get(UATKeychainItemKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// GetActiveContext reads the stored OAuth active context from the keyring.
// Returns nil, nil when no active context has been saved yet.
func GetActiveContext() (*ActiveContext, error) {
	if KeyRing == nil {
		return nil, nil
	}
	data, err := KeyRing.Get(OAuthActiveContextKeychainKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var ac ActiveContext
	if err := json.Unmarshal(data, &ac); err != nil {
		return nil, err
	}
	return &ac, nil
}

// SaveActiveContext persists the active OAuth context (account ID + livemode) in
// the keyring so that ResolveCredentials can build the Stripe-Context header.
func SaveActiveContext(accountID string, livemode bool) error {
	if KeyRing == nil {
		return nil
	}
	data, err := json.Marshal(ActiveContext{AccountID: accountID, Livemode: livemode})
	if err != nil {
		return err
	}
	return KeyRing.Set(OAuthActiveContextKeychainKey, data, "Stripe CLI OAuth active context")
}

// SaveUATExpiresAt persists the UAT expiry time in the keyring.
func SaveUATExpiresAt(t time.Time) error {
	if KeyRing == nil {
		return nil
	}
	return KeyRing.Set(OAuthUATExpiresAtKeychainKey, []byte(t.UTC().Format(time.RFC3339)), "Stripe CLI OAuth token expiry")
}

// GetUATExpiresAt retrieves the stored UAT expiry time from the keyring.
// Returns ErrKeyNotFound (wrapped) when no expiry has been saved.
func GetUATExpiresAt() (time.Time, error) {
	if KeyRing == nil {
		return time.Time{}, keyring.ErrKeyNotFound
	}
	data, err := KeyRing.Get(OAuthUATExpiresAtKeychainKey)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, string(data))
}

// PrintActiveContextBanner prints the active context to stderr once per
// process. Call this at the start of commands that make user-visible Stripe
// API requests (resource commands, raw HTTP, fixtures, triggers).
func (p *Profile) PrintActiveContextBanner() {
	printActiveContextOnce.Do(func() {
		uat, _ := p.GetUAT()
		if !strings.HasPrefix(uat, "oak_") {
			return
		}
		ac, _ := GetActiveContext()
		if ac == nil {
			return
		}
		mode := "sandbox"
		if ac.Livemode {
			mode = "live"
		}
		color := ansi.Color(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s Running in %s · %s (%s)\n", color.Faint("▸"), p.GetDisplayName(), mode, ac.AccountID)
	})
}

// GetUserInfo reads the stored UserInfo from the profile config.
// Returns nil, nil when no user_info has been saved yet.
// TODO: remove with legacy RAK/OIDC flow.
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

// GetCompartmentID returns the account ID for the given livemode from the
// legacy OIDC UserInfo stored in the config file.
// TODO: remove with legacy RAK/OIDC flow.
func (p *Profile) GetCompartmentID(livemode bool) (string, error) {
	ui, err := p.GetUserInfo()
	if err != nil || ui == nil {
		return "", err
	}
	for _, c := range ui.Compartments {
		if c.Livemode == livemode {
			return c.CompartmentID, nil
		}
	}
	return "", nil
}

// ActiveContextLivemodeMismatchError indicates that the requested livemode
// does not match the OAuth active context's livemode. Callers that expose
// their own mode selection (e.g. a --live flag) can catch this with
// errors.As and explain how to reconcile it; callers that don't care which
// mode is used can retry ResolveCredentials(err.ActiveLivemode) to get
// credentials for whichever mode is actually active.
type ActiveContextLivemodeMismatchError struct {
	RequestedLivemode bool
	ActiveLivemode    bool
}

func (e *ActiveContextLivemodeMismatchError) Error() string {
	if e.ActiveLivemode {
		return "You're in live mode. Run 'stripe switch context' to select a sandbox."
	}
	return "You're in a sandbox. Run 'stripe switch context' to select a live account."
}

// ResolveCredentials returns the credentials for the given mode. If an OAK
// token (prefix "oak_") is stored in the keyring and no explicit override is
// active, it is preferred over the configured API key. For OAK tokens the
// active context stored in the keyring sets both Stripe-Context and
// Stripe-Livemode; the livemode parameter is used only for the legacy OIDC
// fallback and plain API key path. If the active context's livemode differs
// from the requested livemode, it returns an *ActiveContextLivemodeMismatchError.
func (p *Profile) ResolveCredentials(livemode bool) (stripe.Credentials, error) {
	if !p.HasOverrideAPIKey() {
		uat, err := p.GetUAT()
		if err != nil {
			return stripe.Credentials{}, err
		}
		if strings.HasPrefix(uat, "oak_") {
			if OAuthTokenRefresher != nil {
				if t, tErr := GetUATExpiresAt(); tErr == nil && time.Until(t) < 60*time.Second {
					refreshMu.Lock()
					// Re-check after acquiring the lock; another goroutine may have
					// already refreshed, bumping the expiry forward.
					if t2, tErr2 := GetUATExpiresAt(); tErr2 == nil && time.Until(t2) < 60*time.Second {
						if refreshErr := OAuthTokenRefresher(p); refreshErr != nil {
							refreshMu.Unlock()
							return stripe.Credentials{}, refreshErr
						}
						uat = p.UAT
					}
					refreshMu.Unlock()
				}
			}
			ac, err := GetActiveContext()
			if err != nil {
				return stripe.Credentials{}, err
			}
			if ac != nil {
				if ac.Livemode != livemode {
					return stripe.Credentials{}, errorcategory.With(&ActiveContextLivemodeMismatchError{
						RequestedLivemode: livemode,
						ActiveLivemode:    ac.Livemode,
					}, errorcategory.UserInput)
				}
				return stripe.NewOAKCredentials(uat, ac.AccountID, livemode), nil
			}
			// TODO: remove with legacy RAK/OIDC flow.
			compartmentID, err := p.GetCompartmentID(livemode)
			if err != nil {
				return stripe.Credentials{}, err
			}
			if livemode && compartmentID == "" {
				return stripe.Credentials{}, errorcategory.UserInputErrorf("You're logged in to a sandbox. To access live mode, reauthenticate with 'stripe login' and select your live account.")
			}
			return stripe.NewOAKCredentials(uat, compartmentID, livemode), nil
		}
	}
	key, err := p.GetAPIKey(livemode)
	if err != nil {
		return stripe.Credentials{}, err
	}
	return stripe.NewAPIKeyCredentials(key), nil
}

// ResolveCredentialsForAnyMode resolves credentials for specified mode, but if that
// doesn't match the OAuth active context, resolves credentials for whichever
// mode is actually active instead of failing.
func (p *Profile) ResolveCredentialsForAnyMode(livemode bool) (stripe.Credentials, error) {
	creds, err := p.ResolveCredentials(livemode)
	var mismatch *ActiveContextLivemodeMismatchError
	if errors.As(err, &mismatch) {
		return p.ResolveCredentials(mismatch.ActiveLivemode)
	}
	return creds, err
}

// GetSessionCredentials retrieves the session credentials from the keyring
func (p *Profile) GetSessionCredentials() (*SessionCredentials, error) {
	key := p.GetConfigField("stripe_cli_session")
	data, err := KeyRing.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, errorcategory.New(errorcategory.Auth, "no session")
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
		return nil, errorcategory.New(errorcategory.Auth, "found a session, but it doesn't match your current account")
	}

	return &creds, nil
}
