package autoupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestIsOptedOut_ConfigSetFalse(t *testing.T) {
	t.Setenv("STRIPE_NO_AUTO_UPDATE", "")
	configDir := t.TempDir()
	stripeDir := filepath.Join(configDir, "stripe")
	os.MkdirAll(stripeDir, 0755)
	os.WriteFile(filepath.Join(stripeDir, "config.toml"), []byte("[settings]\nauto_update = false\n"), 0644)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	assert.True(t, IsOptedOut())
}

func TestIsOptedOut_ConfigSetTrue(t *testing.T) {
	t.Setenv("STRIPE_NO_AUTO_UPDATE", "")
	configDir := t.TempDir()
	stripeDir := filepath.Join(configDir, "stripe")
	os.MkdirAll(stripeDir, 0755)
	os.WriteFile(filepath.Join(stripeDir, "config.toml"), []byte("[settings]\nauto_update = true\n"), 0644)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	assert.False(t, IsOptedOut())
}

func TestIsCurlInstall_EnvOverride(t *testing.T) {
	t.Setenv("STRIPE_INSTALL_METHOD", "curl")
	assert.True(t, IsCurlInstall())

	t.Setenv("STRIPE_INSTALL_METHOD", "homebrew")
	assert.False(t, IsCurlInstall())
}
