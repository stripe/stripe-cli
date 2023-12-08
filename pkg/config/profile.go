package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

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
}

// config key names
const (
	AccountIDName              = "account_id"
	DeviceNameName             = "device_name"
	DisplayNameName            = "display_name"
	IsTermsAcceptanceValidName = "is_terms_acceptance_valid"
	TestModeAPIKeyName         = "test_mode_api_key"
	TestModePubKeyName         = "test_mode_pub_key"
	TestModeKeyExpiresAtName   = "test_mode_key_expires_at"
	LiveModeAPIKeyName         = "live_mode_api_key"
	LiveModePubKeyName         = "live_mode_pub_key"
	LiveModeKeyExpiresAtName   = "live_mode_key_expires_at"
)

const (
	// DateStringFormat is the format for expiredAt date
	DateStringFormat = "2006-01-02"

	// KeyValidInDays is the number of days the API key is valid for
	KeyValidInDays = 90

	// KeyManagementService is the key management service name
	KeyManagementService = "StripeCLI"
)

// KeyRing ...
var KeyRing keyring.Keyring

// CreateProfile creates a profile when logging in
func (p *Profile) CreateProfile() error {
	// Remove all keys under existing profile first
	v := p.deleteProfile(viper.GetViper())

	writeErr := p.writeProfile(v)
	if writeErr != nil {
		return writeErr
	}

	return nil
}

func (p *Profile) deleteProfile(v *viper.Viper) *viper.Viper {
	for _, key := range v.AllKeys() {
		if strings.HasPrefix(key, p.ProfileName+".") {
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
			return "", err
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
	return viper.WriteConfig()
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
		p.saveLivemodeValue(LiveModeAPIKeyName, strings.TrimSpace(p.LiveModeAPIKey), "Live mode API key")
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

	runtimeViper.MergeInConfig()

	// Do this after we merge the old configs in
	if p.TestModeAPIKey != "" {
		runtimeViper = p.safeRemove(runtimeViper, "secret_key")
		runtimeViper = p.safeRemove(runtimeViper, "api_key")
	}

	if p.TestModePublishableKey != "" {
		runtimeViper = p.safeRemove(runtimeViper, "publishable_key")
	}

	runtimeViper.SetConfigFile(profilesFile)

	// Ensure we preserve the config file type
	runtimeViper.SetConfigType(filepath.Ext(profilesFile))

	err = runtimeViper.WriteConfig()
	if err != nil {
		return err
	}

	return nil
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
func (p *Profile) saveLivemodeValue(field, value, description string) {
	fieldID := p.GetConfigField(field)
	_ = KeyRing.Set(keyring.Item{
		Key:         fieldID,
		Data:        []byte(value),
		Description: description,
		Label:       fieldID,
	})
}

// retrieveLivemodeValue retrieves livemode value of given key in keyring
func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
	fieldID := p.GetConfigField(key)
	existingKeys, err := KeyRing.Keys()
	if err != nil {
		return "", err
	}

	for _, item := range existingKeys {
		if item == fieldID {
			value, _ := KeyRing.Get(fieldID)
			return string(value.Data), nil
		}
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// deleteLivemodeValue deletes livemode value of given key in keyring
func (p *Profile) deleteLivemodeValue(key string) error {
	fieldID := p.GetConfigField(key)
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

// ExperimentalFields are currently only used for request signing
type ExperimentalFields struct {
	ContextualName string
	PrivateKey     string
	StripeHeaders  string
}

const (
	experimentalPrefix         = "experimental"
	experimentalStripeHeaders  = experimentalPrefix + "." + "stripe_headers"
	experimentalContextualName = experimentalPrefix + "." + "contextual_name"
	experimentalPrivateKey     = experimentalPrefix + "." + "private_key"
)

// GetExperimentalFields returns a struct of the profile's experimental fields. These fields are only ever additive in functionality.
// If the API key is being overridden, via the --api-key flag or STRIPE_API_KEY env variable, this returns an empty struct.
func (p *Profile) GetExperimentalFields() ExperimentalFields {
	if err := viper.ReadInConfig(); err == nil && os.Getenv("STRIPE_API_KEY") == "" && p.APIKey == "" {
		name := viper.GetString(p.GetConfigField(experimentalContextualName))
		privKey := viper.GetString(p.GetConfigField(experimentalPrivateKey))
		headers := viper.GetString(p.GetConfigField(experimentalStripeHeaders))

		return ExperimentalFields{
			ContextualName: name,
			PrivateKey:     privKey,
			StripeHeaders:  headers,
		}
	}
	return ExperimentalFields{
		ContextualName: "",
		PrivateKey:     "",
		StripeHeaders:  "",
	}
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
	ring, err := keyring.Open(keyring.Config{
		KeychainTrustApplication: true,
		ServiceName:              KeyManagementService,
	})
	if err != nil {
		return nil, err
	}
	keyringItem, err := ring.Get(key)
	if err != nil {
		if err == keyring.ErrKeyNotFound {
			return nil, errors.New("no session")
		}
		return nil, err
	}

	creds := SessionCredentials{}
	if err := json.Unmarshal(keyringItem.Data, &creds); err != nil {
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
