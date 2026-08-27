package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAutoUpdateEnabledDefaultsToTrue(t *testing.T) {
	viper.Reset()

	defer viper.Reset()

	assert.True(t, AutoUpdateEnabled(), "a config file that says nothing opts in")
}

func TestAutoUpdateEnabledReadsTheSettingsTable(t *testing.T) {
	viper.Reset()

	defer viper.Reset()

	viper.Set(AutoUpdateSettingKey(), false)
	assert.False(t, AutoUpdateEnabled())

	viper.Set(AutoUpdateSettingKey(), true)
	assert.True(t, AutoUpdateEnabled())
}

func TestAutoUpdateEnabledReadsTheFlatKey(t *testing.T) {
	viper.Reset()

	defer viper.Reset()

	// The design note showed a bare top-level key, so some users will have written
	// that instead of the [settings] form the install script prints.
	viper.Set(AutoUpdateField, false)
	assert.False(t, AutoUpdateEnabled())
}

func TestAutoUpdateEnabledPrefersTheSettingsTable(t *testing.T) {
	viper.Reset()

	defer viper.Reset()

	viper.Set(AutoUpdateSettingKey(), true)
	viper.Set(AutoUpdateField, false)
	assert.True(t, AutoUpdateEnabled(), "the documented spelling wins when both are present")
}

func TestSettingsIsNotRemovableAsAProfile(t *testing.T) {
	assert.True(t, isReservedConfigKey(SettingsTableName),
		"a profile named settings must not be able to delete machine-wide settings")
}
