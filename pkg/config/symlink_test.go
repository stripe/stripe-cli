package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestConfigWriteConfigFieldRefusesSymlink(t *testing.T) {
	profilesFile, victimFile := setupSymlinkedProfilesFile(t)

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile: Profile{
			ProfileName: "default",
		},
	}
	c.InitConfig()
	KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

	err := c.WriteConfigField("default.color", "on")
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "original = true\n", string(victimContents))
}

func TestProfileWriteConfigFieldRefusesSymlink(t *testing.T) {
	profilesFile, victimFile := setupSymlinkedProfilesFile(t)

	p := Profile{ProfileName: "default"}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile:      p,
	}
	c.InitConfig()
	KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

	err := p.WriteConfigField("color", "on")
	require.ErrorContains(t, err, "symlink")

	victimContents, err := os.ReadFile(victimFile)
	require.NoError(t, err)
	require.Equal(t, "original = true\n", string(victimContents))
}

func TestConfigWriteConfigFieldRefusesSymlinkedParent(t *testing.T) {
	profilesFile, victimDir := setupProfilesFileWithSymlinkedParent(t)

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile: Profile{
			ProfileName: "default",
		},
	}
	c.InitConfig()
	KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

	err := c.WriteConfigField("default.color", "on")
	require.ErrorContains(t, err, "symlink")

	_, err = os.Stat(filepath.Join(victimDir, "config.toml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestWriteProfileRefusesSymlinkedParent(t *testing.T) {
	profilesFile, victimDir := setupProfilesFileWithSymlinkedParent(t)

	p := Profile{
		ProfileName:    "default",
		DeviceName:     "test-device",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: profilesFile,
		Profile:      p,
	}
	c.InitConfig()
	KeyRing = keyring.NewArrayKeyring([]keyring.Item{})

	v := viper.New()
	v.SetConfigFile(profilesFile)

	err := p.writeProfile(v)
	require.ErrorContains(t, err, "symlink")

	_, err = os.Stat(filepath.Join(victimDir, "config.toml"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func setupSymlinkedProfilesFile(t *testing.T) (string, string) {
	t.Helper()
	t.Cleanup(viper.Reset)

	tempDir := t.TempDir()
	victimFile := filepath.Join(tempDir, "victim.toml")
	require.NoError(t, os.WriteFile(victimFile, []byte("original = true\n"), 0o600))

	profilesFile := filepath.Join(tempDir, "config.toml")
	require.NoError(t, os.Symlink(victimFile, profilesFile))

	return profilesFile, victimFile
}

func setupProfilesFileWithSymlinkedParent(t *testing.T) (string, string) {
	t.Helper()
	t.Cleanup(viper.Reset)

	tempDir := t.TempDir()
	victimDir := filepath.Join(tempDir, "victim-dir")
	require.NoError(t, os.MkdirAll(victimDir, 0o755))

	configDir := filepath.Join(tempDir, "config-link")
	require.NoError(t, os.Symlink(victimDir, configDir))

	return filepath.Join(configDir, "config.toml"), victimDir
}
