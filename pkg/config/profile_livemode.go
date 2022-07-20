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
func (p *Profile) RetrieveLivemodeValue(key string) string {
	fieldID := p.GetConfigField(key)
	value, _ := KeyRing.Get(fieldID)
	return string(value.Data)
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
