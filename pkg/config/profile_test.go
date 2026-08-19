package config

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
	"github.com/stripe/stripe-cli/pkg/keyring"
)

// setupProfileConfig writes contents to a temp config file and initializes the
// global viper from it, so tests exercise the same read path as the CLI rather
// than viper's override layer, which nests keys by splitting on ".".
func setupProfileConfig(t *testing.T, contents string) (*Config, string) {
	t.Helper()

	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte(contents), 0600))

	if KeyRing == nil {
		KeyRing = keyring.NewMemoryStore(nil)
	}
	t.Cleanup(func() {
		KeyRing = nil
		viper.Reset()
	})

	c := &Config{LogLevel: "info", ProfilesFile: profilesFile}
	c.InitConfig()

	return c, profilesFile
}

func TestWarnIfLegacyProfileName(t *testing.T) {
	tests := []struct {
		name        string
		profileName string
		contents    string
		wantWarning bool
	}{
		{
			name:        "existing dotted profile warns",
			profileName: "example.project",
			contents:    "[\"example.project\"]\ndisplay_name = 'Legacy'\n",
			wantWarning: true,
		},
		{
			// Nothing is broken until such a profile is actually written, so
			// warning here would only be noise.
			name:        "dotted profile that is not in the config file is silent",
			profileName: "example.project",
			contents:    "[default]\ndisplay_name = 'Default'\n",
		},
		{
			name:        "valid profile is silent",
			profileName: "default",
			contents:    "[default]\ndisplay_name = 'Default'\n",
		},
		{
			name:     "empty profile name is silent",
			contents: "[default]\ndisplay_name = 'Default'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupProfileConfig(t, tt.contents)

			warnLegacyProfileNameOnce = sync.Once{}
			t.Cleanup(func() { warnLegacyProfileNameOnce = sync.Once{} })

			p := Profile{ProfileName: tt.profileName}
			stderr := captureStderr(t, p.WarnIfLegacyProfileName)

			if !tt.wantWarning {
				require.Empty(t, stderr)
				return
			}

			require.Contains(t, stderr, `The profile "example.project" contains a period`)
			require.Contains(t, stderr, "stripe login --project-name example-project")
			require.Contains(t, stderr, "stripe config --remove-profile example.project")
		})
	}
}

// The warning is printed once per process so it does not repeat for every
// command in a session.
func TestWarnIfLegacyProfileNameOnlyWarnsOnce(t *testing.T) {
	setupProfileConfig(t, "[\"example.project\"]\ndisplay_name = 'Legacy'\n")

	warnLegacyProfileNameOnce = sync.Once{}
	t.Cleanup(func() { warnLegacyProfileNameOnce = sync.Once{} })

	p := Profile{ProfileName: "example.project"}
	require.NotEmpty(t, captureStderr(t, p.WarnIfLegacyProfileName))
	require.Empty(t, captureStderr(t, p.WarnIfLegacyProfileName))
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	require.NoError(t, err)

	original := os.Stderr
	os.Stderr = w
	fn()
	os.Stderr = original
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(out)
}

func TestWriteProfile(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		DeviceName:     "st-testing",
		ProfileName:    "tests",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	v := viper.New()

	err := p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expiresAt := getKeyExpiresAt()
	expectedConfig := `[tests]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'
`

	require.EqualValues(t, expectedConfig, string(configValues))
}

func TestWriteProfilesMerge(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		ProfileName:    "tests",
		DeviceName:     "st-testing",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "test-account-display-name",
	}

	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	v := viper.New()
	writeErr := p.writeProfile(v)

	require.NoError(t, writeErr)
	require.FileExists(t, c.ProfilesFile)

	p.ProfileName = "tests-merge"
	writeErrTwo := p.writeProfile(v)
	require.NoError(t, writeErrTwo)
	require.FileExists(t, c.ProfilesFile)

	configValues := helperLoadBytes(t, c.ProfilesFile)
	expiresAt := getKeyExpiresAt()
	expectedConfig := `[tests]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'

[tests-merge]
device_name = 'st-testing'
display_name = 'test-account-display-name'
test_mode_api_key = 'sk_test_123'
test_mode_key_expires_at = '` + expiresAt + `'
`

	require.EqualValues(t, expectedConfig, string(configValues))
}

func TestOldProfileDeleted(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		ProfileName:    "test",
		DeviceName:     "device-before-test",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "display-name-before-test",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	p.WriteConfigField("experimental.stripe_headers", "test-headers")
	p.WriteConfigField("color", "on")

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	untouchedProfile := Profile{
		ProfileName:    "foo",
		DeviceName:     "foo-device-name",
		TestModeAPIKey: "foo_test_123",
	}
	err = untouchedProfile.writeProfile(v)
	require.NoError(t, err)

	p = Profile{
		ProfileName:    "test",
		DeviceName:     "device-after-test",
		TestModeAPIKey: "sk_test_456",
		DisplayName:    "",
	}

	v = p.deleteAuthFields(v)
	err = p.writeProfile(v)
	require.NoError(t, err)

	require.FileExists(t, c.ProfilesFile)

	// Overwrites auth keys
	require.Equal(t, "device-after-test", v.GetString(p.GetConfigField(DeviceNameName)))
	require.Equal(t, "sk_test_456", v.GetString(p.GetConfigField(TestModeAPIKeyName)))
	require.Equal(t, "", v.GetString(p.GetConfigField(DisplayNameName)))
	// Deletes experimental section
	require.False(t, v.IsSet(v.GetString(p.GetConfigField("experimental.stripe_headers"))))
	require.False(t, v.IsSet(v.GetString(p.GetConfigField("experimental"))))
	// Leaves non-auth fields untouched
	require.Equal(t, "on", v.GetString(p.GetConfigField("color")))
	// Leaves the other profile untouched
	require.Equal(t, "foo-device-name", v.GetString(untouchedProfile.GetConfigField(DeviceNameName)))
	require.Equal(t, "foo_test_123", v.GetString(untouchedProfile.GetConfigField(TestModeAPIKeyName)))
}

func TestLiveModeAPIKeyKeychainItemDeleted(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		ProfileName:    "test",
		DeviceName:     "device-before-test",
		LiveModeAPIKey: "",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "display-name-before-test",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		"test.live_mode_api_key": []byte("rk_live_0000000001"),
	})
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	err = p.CreateProfile()
	require.NoError(t, err)

	_, err = KeyRing.Get("test.live_mode_api_key")
	require.Equal(t, keyring.ErrKeyNotFound, err)
}

func TestLiveModeAPIKeyKeychainItemCreated(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		ProfileName:    "test",
		DeviceName:     "device-before-test",
		LiveModeAPIKey: "rk_live_0000000001",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "display-name-before-test",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(nil)
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	err = p.CreateProfile()
	require.NoError(t, err)

	data, err := KeyRing.Get("test.live_mode_api_key")
	require.NoError(t, err)
	require.Equal(t, []byte("rk_live_0000000001"), data)
}

func TestLiveModeAPIKeyKeychainItemReplaced(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	p := Profile{
		ProfileName:    "test",
		DeviceName:     "device-before-test",
		LiveModeAPIKey: "rk_live_0000000002",
		TestModeAPIKey: "sk_test_123",
		DisplayName:    "display-name-before-test",
	}
	c := &Config{
		Color:        "auto",
		LogLevel:     "info",
		Profile:      p,
		ProfilesFile: profilesFile,
	}
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		"test.live_mode_api_key": []byte("rk_live_0000000001"),
	})
	t.Cleanup(func() { KeyRing = nil })
	c.InitConfig()

	v := viper.New()

	v.SetConfigFile(profilesFile)
	err := p.writeProfile(v)
	require.NoError(t, err)

	err = p.CreateProfile()
	require.NoError(t, err)

	data, err := KeyRing.Get("test.live_mode_api_key")
	require.NoError(t, err)
	require.Equal(t, []byte("rk_live_0000000002"), data)
}

func TestResolveCredentialsOAKLivemodeNoCompartment(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte{}, 0600))
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		UATKeychainItemKey: []byte("oak_live_1234567890"),
	})
	t.Cleanup(func() {
		KeyRing = nil
		viper.Reset()
	})
	(&Config{LogLevel: "info", ProfilesFile: profilesFile}).InitConfig()

	p := Profile{ProfileName: "default"}
	_, err := p.ResolveCredentials(true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "You're logged in to a sandbox")
	require.Contains(t, err.Error(), "stripe login")
}

func TestResolveCredentialsOAKLivemodeWithCompartment(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte{}, 0600))
	activeCtxJSON, err := json.Marshal(ActiveContext{AccountID: "acct_live123", Livemode: true})
	require.NoError(t, err)
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		UATKeychainItemKey:            []byte("oak_live_1234567890"),
		OAuthActiveContextKeychainKey: activeCtxJSON,
	})
	t.Cleanup(func() {
		KeyRing = nil
		viper.Reset()
	})
	(&Config{LogLevel: "info", ProfilesFile: profilesFile}).InitConfig()

	p := Profile{ProfileName: "default"}
	creds, err := p.ResolveCredentials(true)
	require.NoError(t, err)
	require.Equal(t, "oak_live_1234567890", creds.Token)
}

func TestResolveCredentialsOAKLivemodeMismatchReturnsTypedError(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte{}, 0600))
	activeCtxJSON, err := json.Marshal(ActiveContext{AccountID: "acct_live123", Livemode: true})
	require.NoError(t, err)
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		UATKeychainItemKey:            []byte("oak_live_1234567890"),
		OAuthActiveContextKeychainKey: activeCtxJSON,
	})
	t.Cleanup(func() {
		KeyRing = nil
		viper.Reset()
	})
	(&Config{LogLevel: "info", ProfilesFile: profilesFile}).InitConfig()

	p := Profile{ProfileName: "default"}
	_, err = p.ResolveCredentials(false)
	require.Error(t, err)

	var mismatch *ActiveContextLivemodeMismatchError
	require.ErrorAs(t, err, &mismatch)
	require.False(t, mismatch.RequestedLivemode)
	require.True(t, mismatch.ActiveLivemode)

	category, ok := errorcategory.Get(err)
	require.True(t, ok)
	require.Equal(t, errorcategory.UserInput, category)

	// Retrying with the active context's actual mode succeeds.
	creds, err := p.ResolveCredentials(mismatch.ActiveLivemode)
	require.NoError(t, err)
	require.Equal(t, "oak_live_1234567890", creds.Token)
}

func TestResolveCredentialsForAnyModeRetriesOnLivemodeMismatch(t *testing.T) {
	profilesFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(profilesFile, []byte{}, 0600))
	activeCtxJSON, err := json.Marshal(ActiveContext{AccountID: "acct_live123", Livemode: true})
	require.NoError(t, err)
	KeyRing = keyring.NewMemoryStore(map[string][]byte{
		UATKeychainItemKey:            []byte("oak_live_1234567890"),
		OAuthActiveContextKeychainKey: activeCtxJSON,
	})
	t.Cleanup(func() {
		KeyRing = nil
		viper.Reset()
	})
	(&Config{LogLevel: "info", ProfilesFile: profilesFile}).InitConfig()

	p := Profile{ProfileName: "default"}
	creds, err := p.ResolveCredentialsForAnyMode(false)
	require.NoError(t, err)
	require.Equal(t, "oak_live_1234567890", creds.Token)
}

func helperLoadBytes(t *testing.T, name string) []byte {
	bytes, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}

	return bytes
}
