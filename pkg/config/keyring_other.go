//go:build !darwin

package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	zkr "github.com/zalando/go-keyring"
)

type zalandoStore struct {
	service string
}

func (s *zalandoStore) Get(key string) ([]byte, error) {
	val, err := zkr.Get(s.service, key)
	if err == zkr.ErrNotFound {
		return nil, ErrKeyNotFound
	}
	if err != nil {
		return nil, err
	}
	return []byte(val), nil
}

func (s *zalandoStore) Set(key string, data []byte, description string) error {
	return zkr.Set(s.service, key, string(data))
}

func (s *zalandoStore) Remove(key string) error {
	err := zkr.Delete(s.service, key)
	if err == zkr.ErrNotFound {
		return ErrKeyNotFound
	}
	return err
}

func (s *zalandoStore) Keys() ([]string, error) {
	return []string{}, nil
}

// wslFileStore is an encrypted file-based store for WSL where D-Bus
// (required by zalando/go-keyring on Linux) is not available.
type wslFileStore struct {
	dir string
	key []byte
	mu  sync.Mutex
}

type wslFileData struct {
	Items map[string][]byte `json:"items"`
}

func newWSLFileStore(dir string) (*wslFileStore, error) {
	pass, err := wslFilePassword("")
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(pass))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return &wslFileStore{dir: dir, key: hash[:]}, nil
}

func (s *wslFileStore) path() string {
	return filepath.Join(s.dir, "keys.enc")
}

func (s *wslFileStore) load() (wslFileData, error) {
	data := wslFileData{Items: make(map[string][]byte)}
	raw, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, err
	}
	plaintext, err := s.decrypt(raw)
	if err != nil {
		return data, err
	}
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return wslFileData{Items: make(map[string][]byte)}, nil
	}
	return data, nil
}

func (s *wslFileStore) save(data wslFileData) error {
	plaintext, err := json.Marshal(data)
	if err != nil {
		return err
	}
	ciphertext, err := s.encrypt(plaintext)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(), ciphertext, 0600)
}

func (s *wslFileStore) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (s *wslFileStore) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrKeyNotFound
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ct, nil)
}

func (s *wslFileStore) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	val, ok := data.Items[key]
	if !ok {
		return nil, ErrKeyNotFound
	}
	return val, nil
}

func (s *wslFileStore) Set(key string, value []byte, description string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return err
	}
	data.Items[key] = value
	return s.save(data)
}

func (s *wslFileStore) Remove(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := data.Items[key]; !ok {
		return ErrKeyNotFound
	}
	delete(data.Items, key)
	return s.save(data)
}

func (s *wslFileStore) Keys() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(data.Items))
	for k := range data.Items {
		keys = append(keys, k)
	}
	return keys, nil
}

func newSecureStore() SecureStore {
	if runtime.GOOS == "linux" && isWSL() {
		dir := getConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
		store, err := newWSLFileStore(dir)
		if err == nil {
			return store
		}
	}
	return &zalandoStore{service: KeyManagementService}
}
