package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/keyring"
)

func TestInitConfigDoesNotChmodThroughSymlink(t *testing.T) {
	for _, parent := range []bool{false, true} {
		name := "file"
		if parent {
			name = "parent"
		}
		t.Run(name, func(t *testing.T) {
			var path, target string
			if parent {
				var dir string
				path, dir = setupProfilesFileWithSymlinkedParent(t)
				target = filepath.Join(dir, "config.toml")
			} else {
				path, target = setupSymlinkedProfilesFile(t)
			}
			original := []byte("original = true\n")
			require.NoError(t, os.WriteFile(target, original, 0600))
			require.NoError(t, os.Chmod(target, 0644))
			originalChmod := configFileChmod
			t.Cleanup(func() { configFileChmod = originalChmod })
			configFileChmod = func(string, os.FileMode) error {
				t.Error("must not chmod through a symlink")
				return os.ErrPermission
			}
			oldKeyRing := KeyRing
			KeyRing = keyring.NewMemoryStore(nil)
			t.Cleanup(func() { KeyRing = oldKeyRing })
			c := &Config{LogLevel: "info", ProfilesFile: path}
			c.InitConfig()
			require.ErrorContains(t, c.WriteConfigField("color", "off"), "symlink")
			contents, err := os.ReadFile(target)
			require.NoError(t, err)
			require.Equal(t, original, contents)
			info, err := os.Stat(target)
			require.NoError(t, err)
			if runtime.GOOS != "windows" {
				require.Equal(t, os.FileMode(0644), info.Mode().Perm())
			}
		})
	}
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
	KeyRing = keyring.NewMemoryStore(nil)

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
	KeyRing = keyring.NewMemoryStore(nil)

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
	KeyRing = keyring.NewMemoryStore(nil)

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
