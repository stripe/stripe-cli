package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestListCmdPrintsAvailablePluginsAfterRefresh(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	lc := NewListCmd(cfg)
	lc.fs = fs
	lc.refreshManifest = func(ctx context.Context, fs afero.Fs) error {
		return writeListManifest(cfg, fs, testListManifest())
	}

	assertListOutput(t, executeListCmd(t, lc))
}

func TestListCmdFallsBackToCachedPluginsWhenRefreshFails(t *testing.T) {
	cfg, fs, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	require.NoError(t, writeListManifest(cfg, fs, testListManifest()))

	lc := NewListCmd(cfg)
	lc.fs = fs
	lc.refreshManifest = func(ctx context.Context, fs afero.Fs) error {
		return errors.New("refresh failed")
	}

	assertListOutput(t, executeListCmd(t, lc))
}

func executeListCmd(t *testing.T, lc *ListCmd) string {
	t.Helper()

	var output bytes.Buffer
	lc.Cmd.SetOut(&output)
	lc.Cmd.SetErr(&output)
	lc.Cmd.SetContext(context.Background())

	require.NoError(t, lc.Cmd.Execute())

	return output.String()
}

func writeListManifest(cfg *config.Config, fs afero.Fs, manifest string) error {
	configPath := cfg.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	manifestPath := filepath.Join(configPath, "plugins.toml")

	if err := fs.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	return afero.WriteFile(fs, manifestPath, []byte(manifest), 0644)
}

func assertListOutput(t *testing.T, rendered string) {
	t.Helper()

	require.Contains(t, rendered, "Available Stripe plugins:")
	requirePluginRow(t, rendered, "apps", "Build and manage Stripe Apps")
	requirePluginRow(t, rendered, "projects", "Scaffold Stripe integration projects")
	requirePluginRow(t, rendered, "tools", "Search internal Stripe operations")
	if runtime.GOOS == "windows" {
		requirePluginRow(t, rendered, "windows-only", "Should be filtered on non-Windows platforms")
	} else {
		require.NotContains(t, rendered, "windows-only")
	}
	require.Contains(t, rendered, "Run `stripe plugin install <name>` to install a plugin.")

	assert.Less(t, strings.Index(rendered, "apps"), strings.Index(rendered, "projects"))
	assert.Less(t, strings.Index(rendered, "projects"), strings.Index(rendered, "tools"))
	if runtime.GOOS == "windows" {
		assert.Less(t, strings.Index(rendered, "tools"), strings.Index(rendered, "windows-only"))
	}
}

func requirePluginRow(t *testing.T, rendered, shortname, shortdesc string) {
	t.Helper()

	require.Regexp(
		t,
		regexp.MustCompile(fmt.Sprintf(`(?m)^  %s\s+%s$`, regexp.QuoteMeta(shortname), regexp.QuoteMeta(shortdesc))),
		rendered,
	)
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
