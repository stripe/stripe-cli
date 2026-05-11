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
