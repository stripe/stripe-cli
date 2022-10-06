//go:build arm64
// +build arm64

package config

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/99designs/keyring"
	"github.com/alessio/shellescape"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
	exec "golang.org/x/sys/execabs"
)

const (
	// DateStringFormat ...
	DateStringFormat = "2006-01-02"

	// KeyValidInDays ...
	KeyValidInDays = 90

	execPathKeychain = "/usr/bin/security"

	// encodingPrefix is a well-known prefix added to strings encoded by Set.
	encodingPrefix = "go-keyring-encoded:"
)

// KeyRing ...
var KeyRing keyring.Keyring

var User = "Stripe CLI"

// saveLivemodeValue saves livemode value of given key in keyring
func (p *Profile) saveLivemodeValue(field, value, description string) {
	// p.WriteConfigField(field, value)

	color := ansi.Color(os.Stdout)
	fmt.Println(color.Yellow(`(!) ON ARM64. WRITING TO GOKEYRING`))

	// err := goKeyring.Set(p.GetConfigField(field), User, value)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	value = encodingPrefix + hex.EncodeToString([]byte(value))

	cmd := exec.Command(execPathKeychain, "-i")
	stdIn, _ := cmd.StdinPipe()

	// save value
	cmd.Start()

	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n", shellescape.Quote(field), shellescape.Quote(User), shellescape.Quote(value))
	io.WriteString(stdIn, command)
	stdIn.Close()
	cmd.Wait()
}

// retrieveLivemodeValue retrieves livemode value of given key in keyring
func (p *Profile) retrieveLivemodeValue(key string) (string, error) {
	fieldID := p.GetConfigField(key)

	color := ansi.Color(os.Stdout)
	fmt.Println(color.Yellow(`(!) ON ARM64. READING FROM GOKEYRING`))

	// value, err := goKeyring.Get(fieldID, User)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// value := viper.GetString(fieldID)

	out, err := exec.Command(
		execPathKeychain,
		"find-generic-password",
		"-s", fieldID,
		"-wa", User).CombinedOutput()
	if err != nil {
		if strings.Contains(string(out), "could not be found") {
			err = fmt.Errorf("KEY NOT FOUND")
		}
		return "", err
	}

	value := strings.TrimSpace(string(out[:]))
	// if the string has the well-known prefix, assume it's encoded
	if strings.HasPrefix(value, encodingPrefix) {
		dec, err := hex.DecodeString(value[len(encodingPrefix):])
		return string(dec), err
	}

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

	if err := viper.ReadInConfig(); err == nil {
		// if the config file has expires at date, then it is using the new livemode key storage
		if viper.IsSet(p.GetConfigField(LiveModeAPIKeyName)) {
			key := viper.GetString(p.GetConfigField(LiveModeAPIKeyName))
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
