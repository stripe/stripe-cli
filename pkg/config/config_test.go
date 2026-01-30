package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/99designs/keyring"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRemoveKey(t *testing.T) {
	v := viper.New()
	v.Set("remove", "me")
	v.Set("stay", "here")

	nv, err := removeKey(v, "remove")
	require.NoError(t, err)

	require.EqualValues(t, []string{"stay"}, nv.AllKeys())
	require.ElementsMatch(t, []string{"stay", "remove"}, v.AllKeys())
}

func setupTestConfig(t *testing.T) (*Config, string, func()) {
	t.Helper()
	profilesFile := filepath.Join(os.TempDir(), "stripe-test", "config.toml")
	os.MkdirAll(filepath.Dir(profilesFile), 0755)

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

	cleanup := func() {
		os.Remove(profilesFile)
		viper.Reset()
	}

	return c, profilesFile, cleanup
}

func TestCopyProfile(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a source profile
	p := Profile{
		ProfileName:    "default",
		DeviceName:     "test-device",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "My Test Account",
		AccountID:      "acct_123",
	}
	c.Profile = p
	err := p.CreateProfile()
	require.NoError(t, err)

	// Re-read config to sync global viper with file
	viper.ReadInConfig()

	// Copy the profile
	err = c.CopyProfile("default", "backup")
	require.NoError(t, err)

	// Verify the backup exists
	v := viper.GetViper()
	require.True(t, v.IsSet("backup"))
	require.Equal(t, "sk_test_123", v.GetString("backup.test_mode_api_key"))
	require.Equal(t, "My Test Account", v.GetString("backup.display_name"))
	require.Equal(t, "backup", v.GetString("backup.profile_name"))
}

func TestCopyProfileErrors(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	// Empty source
	err := c.CopyProfile("", "target")
	require.Error(t, err)
	require.Contains(t, err.Error(), "source profile name cannot be empty")

	// Empty target
	err = c.CopyProfile("source", "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "target profile name cannot be empty")

	// Same source and target
	err = c.CopyProfile("same", "same")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot copy profile to itself")

	// Non-existent source
	err = c.CopyProfile("nonexistent", "target")
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")
}

func TestListProfiles(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create multiple profiles
	profiles := []struct {
		name        string
		displayName string
	}{
		{"default", "Production Account"},
		{"acme corp", "Acme Corp"},
		{"test account", "Test Account"},
	}

	for _, p := range profiles {
		profile := Profile{
			ProfileName:    p.name,
			DeviceName:     "test-device",
			TestModeAPIKey: "sk_test_123",
			DisplayName:    p.displayName,
		}
		err := profile.CreateProfile()
		require.NoError(t, err)
	}

	// Set the active profile
	c.Profile = Profile{ProfileName: "default", DisplayName: "Production Account"}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := c.ListProfiles()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains profile names
	require.Contains(t, output, "Available profiles:")
	require.Contains(t, output, "Acme Corp")
	require.Contains(t, output, "Test Account")
	require.Contains(t, output, "Production Account")
}

func TestListProfilesEmpty(t *testing.T) {
	_, _, cleanup := setupTestConfig(t)
	defer cleanup()

	c := &Config{}

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := c.ListProfiles()
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)
	output := buf.String()

	require.Contains(t, output, "No profiles found")
}

func TestRemoveProfile(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create a profile to remove
	p := Profile{
		ProfileName:    "to-remove",
		DeviceName:     "test-device",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "To Remove",
	}
	err := p.CreateProfile()
	require.NoError(t, err)

	// Verify it exists
	v := viper.GetViper()
	require.True(t, v.IsSet("to-remove"))

	// Remove it
	err = c.RemoveProfile("to-remove")
	require.NoError(t, err)

	// Re-read config and verify it's gone
	c.InitConfig()
	v = viper.GetViper()
	require.False(t, v.IsSet("to-remove"))
}

func TestSwitchProfile(t *testing.T) {
	c, _, cleanup := setupTestConfig(t)
	defer cleanup()

	// Create "default" profile (currently active)
	defaultProfile := Profile{
		ProfileName:    "default",
		DeviceName:     "device-1",
		TestModeAPIKey: "sk_test_default",
		DisplayName:    "Default Account",
		AccountID:      "acct_default",
	}
	c.Profile = defaultProfile
	err := defaultProfile.CreateProfile()
	require.NoError(t, err)

	// Create another profile to switch to (simulating a previous login that was backed up)
	otherProfile := Profile{
		ProfileName:    "other account",
		DeviceName:     "device-2",
		TestModeAPIKey: "sk_test_other",
		DisplayName:    "Other Account",
		AccountID:      "acct_other",
	}
	err = otherProfile.CreateProfile()
	require.NoError(t, err)

	// Capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Switch to the other profile
	err = c.SwitchProfile("other account")
	require.NoError(t, err)

	w.Close()
	os.Stdout = oldStdout
	buf.ReadFrom(r)

	// Verify "default" now has the other account's data
	v := viper.GetViper()
	require.Equal(t, "sk_test_other", v.GetString("default.test_mode_api_key"))
	require.Equal(t, "Other Account", v.GetString("default.display_name"))

	// Verify the previous default was backed up
	require.Equal(t, "sk_test_default", v.GetString("default account.test_mode_api_key"))

	// Verify the old profile key is removed (it was copied to "default" so the original is cleaned up)
	require.False(t, v.IsSet("other account"), "other account should be removed after switch")
}
