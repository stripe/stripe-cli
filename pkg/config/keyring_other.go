//go:build !darwin

package config

import (
	"os"
	"runtime"

	"github.com/99designs/keyring"
)

func getLegacyKeyringConfig() keyring.Config {
	c := keyring.Config{
		KeychainTrustApplication: true,
		ServiceName:              KeyManagementService,
	}

	if runtime.GOOS == "linux" {
		c.FileDir = getConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
		c.FilePasswordFunc = wslFilePassword
		if isWSL() {
			c.AllowedBackends = []keyring.BackendType{keyring.FileBackend}
		} else {
			c.AllowedBackends = []keyring.BackendType{keyring.SecretServiceBackend, keyring.FileBackend}
		}
	}

	return c
}

type legacyKeyringStore struct {
	ring keyring.Keyring
}

func newSecureStore() SecureStore {
	ring, err := keyring.Open(getLegacyKeyringConfig())
	if err != nil {
		return &nullStore{}
	}
	return &legacyKeyringStore{ring: ring}
}

func (s *legacyKeyringStore) Get(key string) ([]byte, error) {
	item, err := s.ring.Get(key)
	if err == keyring.ErrKeyNotFound {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return item.Data, nil
}

func (s *legacyKeyringStore) Set(key string, data []byte, description string) error {
	return s.ring.Set(keyring.Item{
		Key:         key,
		Data:        data,
		Description: description,
		Label:       key,
	})
}

func (s *legacyKeyringStore) Remove(key string) error {
	err := s.ring.Remove(key)
	if err == keyring.ErrKeyNotFound {
		return ErrKeyNotFound
	}
	return err
}

func (s *legacyKeyringStore) Keys() ([]string, error) {
	return s.ring.Keys()
}

// nullStore is a no-op store used when keyring initialization fails.
type nullStore struct{}

func (n *nullStore) Get(string) ([]byte, error)              { return nil, ErrKeyNotFound }
func (n *nullStore) Set(string, []byte, string) error        { return nil }
func (n *nullStore) Remove(string) error                     { return ErrKeyNotFound }
func (n *nullStore) Keys() ([]string, error)                 { return []string{}, nil }
