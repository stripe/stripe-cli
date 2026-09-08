package autoupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOptedOut_EnvVar(t *testing.T) {
	t.Setenv("STRIPE_NO_AUTO_UPDATE", "1")
	assert.True(t, IsOptedOut())
}

func TestIsOptedOut_NoConfig(t *testing.T) {
	t.Setenv("STRIPE_NO_AUTO_UPDATE", "")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	assert.False(t, IsOptedOut())
}

// scripts/install.sh tells users to write a [settings] table; the message the CLI
// prints when it updates tells them to write a top-level auto_update. Whichever
// set of instructions a user followed has to opt them out.
func TestIsOptedOut_Config(t *testing.T) {
	tests := []struct {
		name     string
		config   string
		optedOut bool
	}{
		{"top-level false", "auto_update = false\n", true},
		{"top-level true", "auto_update = true\n", false},
		{"settings table false", "[settings]\nauto_update = false\n", true},
		{"settings table true", "[settings]\nauto_update = true\n", false},
		{
			// Where it lands in a config.toml that already has profiles in it: with
			// the other top-level settings, ahead of the first table.
			"alongside the other top-level settings",
			"color = ''\nauto_update = false\n\n[default]\ndevice_name = 'laptop'\n",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("STRIPE_NO_AUTO_UPDATE", "")
			configDir := t.TempDir()
			stripeDir := filepath.Join(configDir, "stripe")
			require.NoError(t, os.MkdirAll(stripeDir, 0755))
			require.NoError(t, os.WriteFile(filepath.Join(stripeDir, "config.toml"), []byte(tt.config), 0644))
			t.Setenv("XDG_CONFIG_HOME", configDir)

			assert.Equal(t, tt.optedOut, IsOptedOut())
		})
	}
}

func TestIsCurlInstall_EnvOverride(t *testing.T) {
	t.Setenv("STRIPE_INSTALL_METHOD", "curl")
	assert.True(t, IsCurlInstall())

	t.Setenv("STRIPE_INSTALL_METHOD", "homebrew")
	assert.False(t, IsCurlInstall())
}
