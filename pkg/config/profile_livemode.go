package config

import (
	"strings"

	"github.com/99designs/keyring"
)

// DateStringFormat ...
const DateStringFormat = "2006-01-02"

// KeyValidInDays ...
const KeyValidInDays = 90

// KeyRing ...
var KeyRing keyring.Keyring

// saveLivemodeValue saves livemode value of given key in keyring
// func (p *Profile) saveLivemodeValue(field, value, description string) {
// 	fieldID := p.GetConfigField(field)
// 	_ = KeyRing.Set(keyring.Item{
// 		Key:         fieldID,
// 		Data:        []byte(value),
// 		Description: description,
// 		Label:       fieldID,
// 	})
// }

// retrieveLivemodeValue retrieves livemode value of given key in keyring
// func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
// 	fieldID := p.GetConfigField(key)
// 	existingKeys, err := KeyRing.Keys()
// 	if err != nil {
// 		return "", err
// 	}

// 	for _, item := range existingKeys {
// 		if item == fieldID {
// 			value, _ := KeyRing.Get(fieldID)
// 			return string(value.Data), nil
// 		}
// 	}

// 	return "", validators.ErrAPIKeyNotConfigured
// }

// deleteLivemodeValue deletes livemode value of given key in keyring
// func (p *Profile) deleteLivemodeValue(key string) error {
// 	fieldID := p.GetConfigField(key)
// 	existingKeys, err := KeyRing.Keys()
// 	if err != nil {
// 		return err
// 	}
// 	for _, item := range existingKeys {
// 		if item == fieldID {
// 			KeyRing.Remove(fieldID)
// 			return nil
// 		}
// 	}
// 	return nil
// }

// redactAllLivemodeValues redacts all livemode values in the local config file
// func (p *Profile) redactAllLivemodeValues() {
// 	color := ansi.Color(os.Stdout)

// 	if err := viper.ReadInConfig(); err == nil {
// 		// if the config file has expires at date, then it is using the new livemode key storage
// 		if viper.IsSet(p.GetConfigField(LiveModeAPIKeyName)) {
// 			key := viper.GetString(p.GetConfigField(LiveModeAPIKeyName))
// 			if !isRedactedAPIKey(key) {
// 				fmt.Println(color.Yellow(`
// (!) Livemode value found for the field '` + LiveModeAPIKeyName + `' in your config file.
// Livemode values from the config file will be redacted and will not be used.`))

// 				p.WriteConfigField(LiveModeAPIKeyName, RedactAPIKey(key))
// 			}
// 		}
// 	}
// }

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
