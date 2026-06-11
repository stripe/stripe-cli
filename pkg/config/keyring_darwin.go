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

	// Parse dump-keychain output. Each item has "acct" before "svce" in the
	// attributes block, so we collect acct first, then confirm it belongs to
	// our service when we hit the svce line.
	var keys []string
	var currentAcct string
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, `"acct"<blob>=`) {
			currentAcct = extractQuotedValue(trimmed)
		}
		if strings.Contains(trimmed, `"svce"<blob>=`) {
			if extractQuotedValue(trimmed) == k.service && currentAcct != "" {
				keys = append(keys, currentAcct)
			}
			currentAcct = ""
		}
		// Reset on new item boundary
		if strings.HasPrefix(trimmed, "keychain:") {
			currentAcct = ""
		}
	}

	return keys, nil
}

func extractQuotedValue(line string) string {
	start := strings.Index(line, `="`)
	if start == -1 {
		return ""
	}
	start += 2
	end := strings.LastIndex(line, `"`)
	if end <= start {
		return ""
	}
	return line[start:end]
}
