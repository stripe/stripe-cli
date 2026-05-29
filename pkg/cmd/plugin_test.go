package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestPluginCmdRunsListByDefault(t *testing.T) {
	if os.Getenv("BE_TestPluginCmdRunsListByDefault") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=TestPluginCmdRunsListByDefault")
		cmd.Env = append(os.Environ(), "BE_TestPluginCmdRunsListByDefault=1")
		err := cmd.Run()
		require.NoError(t, err)
		return
	}

	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	Config = config.Config{
		Color:        "auto",
		LogLevel:     "info",
		ProfilesFile: filepath.Join(t.TempDir(), "config.toml"),
		Profile: config.Profile{
			ProfileName: "default",
		},
	}
	Config.InitConfig()

	configPath := Config.GetConfigFolder(os.Getenv("XDG_CONFIG_HOME"))
	require.NoError(t, os.MkdirAll(configPath, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(configPath, "plugins.toml"), []byte(testPluginCmdManifest()), 0644))

	pc := newPluginCmd()

	defaultOutput, err := executeCommand(pc.cmd)
	require.NoError(t, err)
	require.Contains(t, defaultOutput, "Available Stripe plugins:")
	require.Contains(t, defaultOutput, "apps      Build and manage Stripe Apps")
	require.Contains(t, defaultOutput, "projects  Scaffold Stripe integration projects")

	listOutput, err := executeCommand(pc.cmd, "list")
	require.NoError(t, err)
	require.Equal(t, defaultOutput, listOutput)
}

func testPluginCmdManifest() string {
	return fmt.Sprintf(`[[Plugin]]
  Shortname = "projects"
  Shortdesc = "Scaffold Stripe integration projects"
  Binary = "stripe-cli-projects"
  MagicCookieValue = "PROJECTS-COOKIE"

  [[Plugin.Release]]
    Arch = "%s"
    OS = "%s"
    Version = "1.0.0"
    Sum = "projects"

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
`, runtime.GOARCH, runtime.GOOS, runtime.GOARCH, runtime.GOOS)
}
