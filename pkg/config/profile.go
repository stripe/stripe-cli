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
)

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

// CreateProfile creates a profile when logging in
func (p *Profile) CreateProfile() error {
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
		key := p.GetConfigField(field)
		if v.IsSet(key) {
			newViper, err := removeKey(v, key)
			if err == nil {
				// failure to remove a key should not break the login flow
				v = newViper
			}
		}
	}
	return v
}

// GetColor gets the color setting for the user based on the flag or the
// persisted color stored in the config file
func (p *Profile) GetColor() (string, error) {
	color := viper.GetString("color")
	if color != "" {
		return color, nil
	}

	color = viper.GetString(p.GetConfigField("color"))
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
		return viper.GetString(p.GetConfigField(DeviceNameName)), nil
	}

	return "", validators.ErrDeviceNameNotConfigured
}

// GetAccountID returns the accountId for the given profile.
func (p *Profile) GetAccountID() (string, error) {
	if p.AccountID != "" {
		return p.AccountID, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString(p.GetConfigField(AccountIDName)), nil
	}

	return "", validators.ErrAccountIDNotConfigured
}

// GetUserID returns the user ID for the given profile.
func (p *Profile) GetUserID() (string, error) {
	if p.UserID != "" {
		return p.UserID, nil
	}

	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString(p.GetConfigField(UserIDName)), nil
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
		if viper.IsSet(p.GetConfigField("secret_key")) {
			p.RegisterAlias(TestModeAPIKeyName, "secret_key")
		} else if viper.IsSet(p.GetConfigField("api_key")) {
			p.RegisterAlias(TestModeAPIKeyName, "api_key")
		}
		return viper.GetString(p.GetConfigField(TestModeAPIKeyName)) != ""
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
		if viper.IsSet(p.GetConfigField("secret_key")) {
			p.RegisterAlias(TestModeAPIKeyName, "secret_key")
		} else if viper.IsSet(p.GetConfigField("api_key")) {
			p.RegisterAlias(TestModeAPIKeyName, "api_key")
		}

		if err := viper.ReadInConfig(); err == nil {
			key = viper.GetString(p.GetConfigField(TestModeAPIKeyName))
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
		timeString = viper.GetString(p.GetConfigField(LiveModeKeyExpiresAtName))
	} else {
		timeString = viper.GetString(p.GetConfigField(TestModeKeyExpiresAtName))
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

		if viper.IsSet(p.GetConfigField("publishable_key")) {
			p.RegisterAlias(TestModePubKeyName, "publishable_key")
		}
		// there is a bug with viper.GetStringMapString when the key name is too long, which makes
		// `config --list --project-name <project_name>` unable to read the project specific config
		if viper.IsSet(p.GetConfigField("test_mode_publishable_key")) {
			p.RegisterAlias(TestModePubKeyName, "test_mode_publishable_key")
		}
	}

	err := viper.ReadInConfig()
	if err != nil {
		return "", err
	}

	key = viper.GetString(p.GetConfigField(fieldID))
	if key != "" {
		return key, nil
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// GetDisplayName returns the account display name of the user
func (p *Profile) GetDisplayName() string {
	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString(p.GetConfigField(DisplayNameName))
	}

	return ""
}

// GetTerminalPOSDeviceID returns the device id from the config for Terminal quickstart to use
func (p *Profile) GetTerminalPOSDeviceID() string {
	if err := viper.ReadInConfig(); err == nil {
		return viper.GetString(p.GetConfigField("terminal_pos_device_id"))
	}

	return ""
}

// GetConfigField returns the configuration field for the specific profile
func (p *Profile) GetConfigField(field string) string {
	return p.ProfileName + "." + field
}

// RegisterAlias registers an alias for a given key.
func (p *Profile) RegisterAlias(alias, key string) {
	viper.RegisterAlias(p.GetConfigField(alias), p.GetConfigField(key))
}

// WriteConfigField updates a configuration field and writes the updated
// configuration to disk.
func (p *Profile) WriteConfigField(field, value string) error {
	viper.ReadInConfig()
	viper.Set(p.GetConfigField(field), value)
	return writeConfig(viper.GetViper())
}

// DeleteConfigField deletes a configuration field.
func (p *Profile) DeleteConfigField(field string) error {
	v, err := removeKey(viper.GetViper(), p.GetConfigField(field))
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

	err := makePath(profilesFile)
	if err != nil {
		return err
	}

	if p.DeviceName != "" {
		runtimeViper.Set(p.GetConfigField(DeviceNameName), strings.TrimSpace(p.DeviceName))
	}

	if p.LiveModeAPIKey != "" {
		expiresAt := getKeyExpiresAt()
		runtimeViper.Set(p.GetConfigField(LiveModeKeyExpiresAtName), expiresAt)

		// // store redacted key in config
		runtimeViper.Set(p.GetConfigField(LiveModeAPIKeyName), RedactAPIKey(strings.TrimSpace(p.LiveModeAPIKey)))

		// // store actual key in secure keyring
		if err := p.saveLivemodeValue(LiveModeAPIKeyName, strings.TrimSpace(p.LiveModeAPIKey), "Live mode API key"); err != nil {
			return err
		}
	}

	if p.LiveModePublishableKey != "" {
		runtimeViper.Set(p.GetConfigField(LiveModePubKeyName), strings.TrimSpace(p.LiveModePublishableKey))
	}

	if p.TestModeAPIKey != "" {
		runtimeViper.Set(p.GetConfigField(TestModeAPIKeyName), strings.TrimSpace(p.TestModeAPIKey))
		runtimeViper.Set(p.GetConfigField(TestModeKeyExpiresAtName), getKeyExpiresAt())
	}

	if p.TestModePublishableKey != "" {
		runtimeViper.Set(p.GetConfigField(TestModePubKeyName), strings.TrimSpace(p.TestModePublishableKey))
	}

	if p.DisplayName != "" {
		runtimeViper.Set(p.GetConfigField(DisplayNameName), strings.TrimSpace(p.DisplayName))
	}

	if p.AccountID != "" {
		runtimeViper.Set(p.GetConfigField(AccountIDName), strings.TrimSpace(p.AccountID))
	}

	if p.UserID != "" {
		runtimeViper.Set(p.GetConfigField(UserIDName), strings.TrimSpace(p.UserID))
	}

	if p.SandboxClaimURL != "" {
		runtimeViper.Set(p.GetConfigField(SandboxClaimURLName), strings.TrimSpace(p.SandboxClaimURL))
	}

	if p.SandboxExpiresAt != "" {
		runtimeViper.Set(p.GetConfigField(SandboxExpiresAtName), strings.TrimSpace(p.SandboxExpiresAt))
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
	if v.IsSet(p.GetConfigField(key)) {
		newViper, err := removeKey(v, p.GetConfigField(key))
		if err == nil {
			// I don't want to fail the entire login process on not being able to remove
			// the old secret_key field so if there's no error
			return newViper
		}
	}

	return v
}

// redactAllLivemodeValues redacts all livemode values in the local config file
func (p *Profile) redactAllLivemodeValues() {
	color := ansi.Color(os.Stdout)

	if err := viper.ReadInConfig(); err == nil {
		// if the config file has expires at date, then it is using the new livemode key storage
		if viper.IsSet(p.GetConfigField(LiveModeAPIKeyName)) {
			key := viper.GetString(p.GetConfigField(LiveModeAPIKeyName))
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
		mode := "test"
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
		return "your active context is livemode; run 'stripe switch context' to select a test mode context"
	}
	return "your active context is test mode; run 'stripe switch context' to select a livemode context"
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
				return stripe.Credentials{}, errorcategory.UserInputErrorf("you're logged in to a sandbox; to access livemode, reauthenticate with 'stripe login' and select your live account")
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
