// Package autoupdate implements automatic version updates for curl-installed Stripe CLI binaries.
package autoupdate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

// IsOptedOut reports whether the user has disabled auto-update.
func IsOptedOut() bool {
	if os.Getenv("STRIPE_NO_AUTO_UPDATE") != "" {
		return true
	}

	configFolder := getConfigFolder()
	configFile := filepath.Join(configFolder, "config.toml")

	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return false
	}

	return !v.GetBool("settings.auto_update") && v.IsSet("settings.auto_update")
}

// IsCurlInstall reports whether the current binary was installed via curl (lives in ~/.stripe/bin/).
func IsCurlInstall() bool {
	if method := os.Getenv("STRIPE_INSTALL_METHOD"); method != "" {
		return method == "curl"
	}

	exe, err := os.Executable()
	if err != nil {
		return false
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return false
	}

	home, err := homedir.Dir()
	if err != nil {
		return false
	}

	stripeBinDir := filepath.Join(home, ".stripe", "bin")
	stripeBinDir, err = filepath.EvalSymlinks(stripeBinDir)
	if err != nil {
		return false
	}

	exeLower := strings.ToLower(filepath.ToSlash(exe))
	expectedLower := strings.ToLower(filepath.ToSlash(stripeBinDir))

	return strings.HasPrefix(exeLower, expectedLower)
}

func getConfigFolder() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "stripe")
	}
	home, err := homedir.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "stripe")
}

// GetStateDirFn is the active implementation; tests can override it.
var GetStateDirFn = getStateDirDefault

// GetStateDir returns the path to the autoupdate state directory.
func GetStateDir() string {
	return GetStateDirFn()
}

func getStateDirDefault() string {
	home, err := homedir.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".stripe", "state")
}
