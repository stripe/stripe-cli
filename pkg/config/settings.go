package config

import (
	"github.com/spf13/viper"
)

const (
	// SettingsTableName is the reserved top-level table holding machine-wide CLI
	// settings that belong to no profile.
	SettingsTableName = "settings"

	// AutoUpdateField is the settings key controlling whether the CLI may replace
	// its own binary.
	AutoUpdateField = "auto_update"
)

// AutoUpdateSettingKey is the config path of the auto-update opt-out, for use
// with WriteConfigField.
func AutoUpdateSettingKey() string {
	return SettingsTableName + "." + AutoUpdateField
}

// AutoUpdateEnabled reports whether the config file allows the CLI to update
// itself. It defaults to true, so a config file that says nothing opts in.
//
// Both the nested and the flat spelling are honored. scripts/install.sh prints
// the [settings] form, but the flat form is what the design note showed and what
// some users will therefore have written; reading only one of them would leave an
// opt-out silently ignored, which is the worst possible failure for this setting.
func AutoUpdateEnabled() bool {
	for _, key := range []string{AutoUpdateSettingKey(), AutoUpdateField} {
		if viper.IsSet(key) {
			return viper.GetBool(key)
		}
	}

	return true
}
