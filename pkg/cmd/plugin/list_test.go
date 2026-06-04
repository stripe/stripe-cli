package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

func TestListCmdPrintsAvailablePlugins(t *testing.T) {
	cfg, _, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	lc := NewListCmd(cfg)
	lc.listPlugins = func(ctx context.Context, cfg config.IConfig, apiBaseURL, dashboardBaseURL string) (plugins.PluginList, error) {
		return testListPluginList(), nil
	}

	assertListOutput(t, executeListCmd(t, lc))
}

func TestListCmdReturnsEndpointErrorWithoutFallback(t *testing.T) {
	cfg, _, cleanup := setupPluginCommandTest(t)
	defer cleanup()

	lc := NewListCmd(cfg)
	lc.listPlugins = func(ctx context.Context, cfg config.IConfig, apiBaseURL, dashboardBaseURL string) (plugins.PluginList, error) {
		return plugins.PluginList{}, errors.New("list failed")
	}

	err := executeListCmdErr(t, lc)
	require.EqualError(t, err, "list failed")
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

func executeListCmdErr(t *testing.T, lc *ListCmd) error {
	t.Helper()

	var output bytes.Buffer
	lc.Cmd.SetOut(&output)
	lc.Cmd.SetErr(&output)
	lc.Cmd.SetContext(context.Background())

	return lc.Cmd.Execute()
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

func testListPluginList() plugins.PluginList {
	return plugins.PluginList{
		Plugins: []plugins.Plugin{
			{
				Shortname: "tools",
				Shortdesc: "Search internal Stripe operations",
				Binary:    "stripe-cli-tools",
				Releases: []plugins.Release{
					{
						Arch:    runtime.GOARCH,
						OS:      runtime.GOOS,
						Version: "1.0.0",
					},
				},
			},
			{
				Shortname: "apps",
				Shortdesc: "Build and manage Stripe Apps",
				Binary:    "stripe-cli-apps",
				Releases: []plugins.Release{
					{
						Arch:    runtime.GOARCH,
						OS:      runtime.GOOS,
						Version: "2.0.0",
					},
				},
			},
			{
				Shortname: "projects",
				Shortdesc: "Scaffold Stripe integration projects",
				Binary:    "stripe-cli-projects",
				Releases: []plugins.Release{
					{
						Arch:    runtime.GOARCH,
						OS:      runtime.GOOS,
						Version: "3.0.0",
					},
				},
			},
			{
				Shortname: "windows-only",
				Shortdesc: "Should be filtered on non-Windows platforms",
				Binary:    "stripe-cli-windows-only",
				Releases: []plugins.Release{
					{
						Arch:    "amd64",
						OS:      "windows",
						Version: "1.0.0",
					},
				},
			},
		},
	}
}
