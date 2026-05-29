package plugin

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCmdPrintsAvailablePlugins(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	manifestPath := filepath.Join(configPath, "plugins.toml")
	require.NoError(t, fs.MkdirAll(configPath, 0755))
	require.NoError(t, afero.WriteFile(fs, manifestPath, []byte(testListManifest()), 0644))

	lc := NewListCmd(cfg)
	lc.fs = fs

	var output bytes.Buffer
	lc.Cmd.SetOut(&output)
	lc.Cmd.SetErr(&output)
	lc.Cmd.SetContext(context.Background())

	require.NoError(t, lc.Cmd.Execute())

	rendered := output.String()
	require.Contains(t, rendered, "Available Stripe plugins:")
	require.Contains(t, rendered, "apps      Build and manage Stripe Apps")
	require.Contains(t, rendered, "projects  Scaffold Stripe integration projects")
	require.Contains(t, rendered, "tools     Search internal Stripe operations")
	require.NotContains(t, rendered, "windows-only")
	require.Contains(t, rendered, "Run `stripe plugin install <name>` to install a plugin.")

	assert.Less(t, strings.Index(rendered, "apps"), strings.Index(rendered, "projects"))
	assert.Less(t, strings.Index(rendered, "projects"), strings.Index(rendered, "tools"))
}

func testListManifest() string {
	return fmt.Sprintf(`[[Plugin]]
  Shortname = "tools"
  Shortdesc = "Search internal Stripe operations"
  Binary = "stripe-cli-tools"
  MagicCookieValue = "TOOLS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "tools"

[[Plugin]]
  Shortname = "apps"
  Shortdesc = "Build and manage Stripe Apps"
  Binary = "stripe-cli-apps"
  MagicCookieValue = "APPS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "2.0.0"
    Sum = "apps"

[[Plugin]]
  Shortname = "projects"
  Shortdesc = "Scaffold Stripe integration projects"
  Binary = "stripe-cli-projects"
  MagicCookieValue = "PROJECTS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "3.0.0"
    Sum = "projects"

[[Plugin]]
  Shortname = "windows-only"
  Shortdesc = "Should be filtered on non-Windows platforms"
  Binary = "stripe-cli-windows-only"
  MagicCookieValue = "WINDOWS-COOKIE"

  [[Plugin.Release]]
    Arch = "amd64"
    OS = "windows"
    Version = "1.0.0"
    Sum = "windows-only"
`, runtime.GOARCH, runtime.GOOS, runtime.GOARCH, runtime.GOOS, runtime.GOARCH, runtime.GOOS)
}
