//go:build darwin

package config

import (
	"encoding/base64"
	"fmt"
	"os/exec"
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
	out, err := exec.Command("/usr/bin/security", "dump-keychain").Output()
	if err != nil {
		return []string{}, nil
	}

	var keys []string
	lines := strings.Split(string(out), "\n")
	var inOurService bool
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `"svce"<blob>=`) {
			inOurService = strings.Contains(trimmed, fmt.Sprintf(`"%s"`, k.service))
		}
		if inOurService && strings.Contains(trimmed, `"acct"<blob>=`) {
			if start := strings.Index(trimmed, `="`); start != -1 {
				start += 2
				if end := strings.LastIndex(trimmed, `"`); end > start {
					keys = append(keys, trimmed[start:end])
				}
			}
		}
	}

	return keys, nil
}
