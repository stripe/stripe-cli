// Package autoupdate implements automatic version updates for curl-installed Stripe CLI binaries.
package autoupdate

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

func isOptedOut() bool {
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

func isCurlInstall() bool {
	if method := os.Getenv("STRIPE_INSTALL_METHOD"); method != "" {
		return method == "curl"
	}

	exe, err := os.Executable()
	if err != nil {
		return false
	}

	home, err := homedir.Dir()
	if err != nil {
		return false
	}

	stripeBinDir := filepath.Join(home, ".stripe", "bin")
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

// getStateDirFn is the active implementation; tests override it.
var getStateDirFn = getStateDirDefault

func getStateDir() string {
	return getStateDirFn()
}

func getStateDirDefault() string {
	home, err := homedir.Dir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".stripe", "state")
}
