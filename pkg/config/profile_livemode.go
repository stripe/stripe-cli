package config

import (
	"strings"

	"github.com/99designs/keyring"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// DateStringFormat ...
const DateStringFormat = "2006-01-02"

// KeyValidInDays ...
const KeyValidInDays = 90

// KeyRing ...
var KeyRing keyring.Keyring

// storeLivemodeValue
func (p *Profile) storeLivemodeValue(field, value, description string) {
	fieldID := p.GetConfigField(field)
	_ = KeyRing.Set(keyring.Item{
		Key:         fieldID,
		Data:        []byte(value),
		Description: description,
		Label:       fieldID,
	})
}

// RetrieveLivemodeValue ...
func (p *Profile) RetrieveLivemodeValue(key string) (string, error) {
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

// DeleteLivemodeValue ...
func (p *Profile) DeleteLivemodeValue(key string) error {
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

// IsRedactedAPIKey ...
func IsRedactedAPIKey(apiKey string) bool {
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
