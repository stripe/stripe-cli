//go:build !arm64
// +build !arm64

package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

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
