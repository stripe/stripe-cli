//go:build darwin

package config

import (
	"encoding/base64"
	"fmt"
	"strings"

	zkr "github.com/zalando/go-keyring"
)

// darwinKeychainService is the service name for keychain items. This differs
// from KeyManagementService ("StripeCLI") used by the old 99designs/keyring
// backend so that we never touch legacy ACL-protected items.
const darwinKeychainService = "stripe-cli"

const encodingPrefix = "b64:"

type darwinKeychain struct {
	service string
}

func newSecureStore() SecureStore {
	return &darwinKeychain{service: darwinKeychainService}
}

func (k *darwinKeychain) Get(key string) ([]byte, error) {
	val, err := zkr.Get(k.service, key)
	if err == zkr.ErrNotFound {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(val, encodingPrefix) {
		decoded, err := base64.StdEncoding.DecodeString(val[len(encodingPrefix):])
		if err != nil {
			return nil, fmt.Errorf("keychain: failed to decode value: %w", err)
		}
		return decoded, nil
	}

	return []byte(val), nil
}

func (k *darwinKeychain) Set(key string, data []byte, description string) error {
	encoded := encodingPrefix + base64.StdEncoding.EncodeToString(data)
	return zkr.Set(k.service, key, encoded)
}

func (k *darwinKeychain) Remove(key string) error {
	err := zkr.Delete(k.service, key)
	if err == zkr.ErrNotFound {
		return ErrKeyNotFound
	}
	return err
}

func (k *darwinKeychain) Keys() ([]string, error) {
	// Not efficiently supported by the security CLI without parsing
	// dump-keychain output. Callers that need to check for a specific key
	// should use Get() directly instead.
	return []string{}, nil
}
