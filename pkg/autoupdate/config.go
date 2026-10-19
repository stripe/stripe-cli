// Package autoupdate implements automatic version updates for Stripe CLI binaries installed with the install script.
package autoupdate

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	"github.com/stripe/stripe-cli/pkg/installmethod"
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

	// Both spellings count. The install script tells users to write a [settings]
	// table; the message the CLI prints when it updates tells them to write a
	// top-level auto_update, which is also where every other CLI setting lives in
	// config.toml. Someone who followed either has said what they want.
	for _, key := range []string{"auto_update", "settings.auto_update"} {
		if v.IsSet(key) && !v.GetBool(key) {
			return true
		}
	}

	return false
}

// IsScriptInstall reports whether the current binary was installed by the install
// script, the only install method that auto-update replaces the binary for.
func IsScriptInstall() bool {
	return installmethod.Detect(installmethod.OSEnv()) == installmethod.Script
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
