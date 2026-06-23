package config

import (
	"os"
	"path/filepath"

	"github.com/stripe/stripe-cli/pkg/keyring"
)

func newSecureStore() keyring.SecureStore {
	return keyring.NewSecureStore(KeyManagementService, CredentialsFilePath())
}

// IsUsingInsecureStorage reports whether credentials have been written to the
// plain-file fallback because the OS keyring was unavailable.
func IsUsingInsecureStorage() bool {
	return keyring.IsUsingInsecureStorage(KeyRing)
}

// CredentialsFilePath returns the path of the plain-text credentials file used
// by the file fallback store.
func CredentialsFilePath() string {
	return filepath.Join(getConfigFolder(os.Getenv("XDG_CONFIG_HOME")), "credentials.json")
}
