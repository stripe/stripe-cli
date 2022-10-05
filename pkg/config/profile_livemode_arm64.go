//go:build arm64
// +build arm64

package config

import (
	"fmt"
	"os"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

// DateStringFormat ...
const DateStringFormat = "2006-01-02"

// KeyValidInDays ...
const KeyValidInDays = 90

// KeyRing ...
var KeyRing keyring.Keyring

// saveLivemodeValue saves livemode value of given key in keyring
func (p *Profile) saveLivemodeValue(field, value, description string) {
	p.WriteConfigField(field, value)
}

// retrieveLivemodeValue retrieves livemode value of given key in keyring
func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
	fieldID := p.GetConfigField(key)
	value := viper.GetString(fieldID)
	if value != "" {
		return value, nil
	}

	return "", validators.ErrAPIKeyNotConfigured
}

// deleteLivemodeValue deletes livemode value of given key in keyring
func (p *Profile) deleteLivemodeValue(key string) error {
	return p.DeleteConfigField(key)
}

// redactAllLivemodeValues redacts all livemode values in the local config file
func (p *Profile) redactAllLivemodeValues() {
	color := ansi.Color(os.Stdout)
	fmt.Println(color.Yellow(`(!) ON ARM64. DO NOTHING`))
}

// RedactAPIKey returns a redacted version of API keys. The first 8 and last 4
// characters are not redacted, everything else is replaced by "*" characters.
//
// It panics if the provided string has less than 12 characters.
func RedactAPIKey(apiKey string) string {
	return apiKey
}

// isRedactedAPIKey checks if the input string is a refacted api key
func isRedactedAPIKey(apiKey string) bool {
	return false
}
