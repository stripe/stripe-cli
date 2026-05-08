package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins"
)

func createPluginCmd() *pluginTemplateCmd {
	plugin := plugins.Plugin{
		Shortname:        "test",
		Shortdesc:        "test your stuff",
		Binary:           "stripe-cli-test",
		MagicCookieValue: "magic",
		Releases: []plugins.Release{{
			Arch:    "amd64",
			OS:      "darwin",
			Version: "0.0.1",
			Sum:     "c53a98c3fa63563227eb8b5601acedb5e0e70fed2e1d52a5918a17ac755f17f7",
		}},
	}

	pluginCmd := newPluginTemplateCmd(&Config, &plugin)

	return pluginCmd
}

// TestFlagsArePassedAsArgs ensures that the plugin is passing all args and flags as expected.
// This is a complex dance between the CLI itself and the plugin, so the flags come from
// two different sources as a result. This test is here to catch any non-obvious regressions
func TestFlagsArePassedAsArgs(t *testing.T) {
	pluginCmd := createPluginCmd()
	rootCmd.AddCommand(pluginCmd.cmd)

	Execute(context.Background())

	// temp override for the os.Args so that the pluginCmd can use them
	oldArgs := os.Args
	os.Args = []string{"stripe", "test", "testarg", "--log-level=info"}
	defer func() { os.Args = oldArgs }()

	rootCmd.SetArgs([]string{"test", "testarg", "--log-level=info"})
	executeCommandC(rootCmd, "test", "testarg", "--log-level=info")

	require.Equal(t, 2, len(pluginCmd.ParsedArgs))
	require.Equal(t, "testarg --log-level=info", strings.Join(pluginCmd.ParsedArgs, " "))
}

func TestAddPluginSubcommandStubs(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "myapp",
		Shortdesc:        "My app plugin",
		Binary:           "stripe-cli-myapp",
		MagicCookieValue: "magic",
		Commands: []plugins.CommandInfo{
			{
				Name: "create",
				Desc: "Create a resource",
			},
			{
				Name: "logs",
				Desc: "View logs",
				Commands: []plugins.CommandInfo{
					{
						Name: "tail",
						Desc: "Tail logs in real-time",
					},
				},
			},
		},
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// Verify subcommand stubs were created
	subCmds := ptc.cmd.Commands()
	require.Equal(t, 2, len(subCmds))

	assert.Equal(t, "create", subCmds[0].Name())
	assert.Equal(t, "Create a resource", subCmds[0].Short)
	assert.Equal(t, "plugin", subCmds[0].Annotations["scope"])

	assert.Equal(t, "logs", subCmds[1].Name())
	assert.Equal(t, "View logs", subCmds[1].Short)

	// Verify nested subcommand
	logSubCmds := subCmds[1].Commands()
	require.Equal(t, 1, len(logSubCmds))
	assert.Equal(t, "tail", logSubCmds[0].Name())
	assert.Equal(t, "Tail logs in real-time", logSubCmds[0].Short)
	assert.Equal(t, "plugin", logSubCmds[0].Annotations["scope"])
}

func TestAddPluginSubcommandStubsEmpty(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "simple",
		Shortdesc:        "A simple plugin",
		Binary:           "stripe-cli-simple",
		MagicCookieValue: "magic",
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// No subcommands should be created
	assert.Equal(t, 0, len(ptc.cmd.Commands()))
}

func TestAddPluginSubcommandStubsSkipsEmptyName(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "badplugin",
		Shortdesc:        "A plugin with bad manifest data",
		Binary:           "stripe-cli-bad",
		MagicCookieValue: "magic",
		Commands: []plugins.CommandInfo{
			{Name: "valid", Desc: "A valid command"},
			{Name: "", Desc: "Entry with empty name"},
			{Name: "also-valid", Desc: "Another valid command"},
		},
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// Only the two valid entries should become subcommands
	cmds := ptc.cmd.Commands()
	assert.Equal(t, 2, len(cmds))
	assert.Equal(t, "also-valid", cmds[0].Name())
	assert.Equal(t, "valid", cmds[1].Name())
}

func TestSubsliceAfter(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		sl       []string
		str      string
	}{
		{"empty slice", []string{}, []string{}, "foo"},
		{"empty string", []string{}, []string{""}, ""},
		{"not found", []string{}, []string{"bar"}, "foo"},
		{"found at beginning", []string{"bar"}, []string{"foo", "bar"}, "foo"},
		{"found at middle", []string{"baz", "qux"}, []string{"foo", "bar", "baz", "qux"}, "bar"},
		{"found at end", []string{}, []string{"foo", "bar"}, "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, subsliceAfter(tt.sl, tt.str))
		})
	}
}

func TestResolvePluginTelemetryCommandPathAddsFirstPluginSubcommand(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	pluginCmd := &cobra.Command{
		Use:         "projects",
		Annotations: map[string]string{"scope": "plugin"},
	}
	root.AddCommand(pluginCmd)

	assert.Equal(
		t,
		"stripe projects catalog",
		resolvePluginTelemetryCommandPath(pluginCmd, []string{"stripe", "projects", "catalog"}),
	)
}

func TestResolvePluginTelemetryCommandPathSkipsPluginGlobalFlags(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	pluginCmd := &cobra.Command{
		Use:         "projects",
		Annotations: map[string]string{"scope": "plugin"},
	}
	root.AddCommand(pluginCmd)

	assert.Equal(
		t,
		"stripe projects catalog",
		resolvePluginTelemetryCommandPath(pluginCmd, []string{"stripe", "projects", "--color", "off", "--project-name", "demo", "catalog"}),
	)
}

func TestResolvePluginTelemetryCommandPathLeavesTopLevelWhenNoSubcommand(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	pluginCmd := &cobra.Command{
		Use:         "projects",
		Annotations: map[string]string{"scope": "plugin"},
	}
	root.AddCommand(pluginCmd)

	assert.Equal(
		t,
		"stripe projects",
		resolvePluginTelemetryCommandPath(pluginCmd, []string{"stripe", "projects", "--help"}),
	)
}

func TestResolvePluginTelemetryCommandPathUsesResolvedPluginStubPath(t *testing.T) {
	root := &cobra.Command{Use: "stripe"}
	pluginCmd := &cobra.Command{
		Use:         "projects",
		Annotations: map[string]string{"scope": "plugin"},
	}
	billingCmd := &cobra.Command{
		Use:         "billing",
		Annotations: map[string]string{"scope": "plugin"},
	}
	listCmd := &cobra.Command{
		Use:         "list",
		Annotations: map[string]string{"scope": "plugin"},
	}

	root.AddCommand(pluginCmd)
	pluginCmd.AddCommand(billingCmd)
	billingCmd.AddCommand(listCmd)

	assert.Equal(
		t,
		"stripe projects billing list",
		resolvePluginTelemetryCommandPath(listCmd, []string{"stripe", "projects", "billing", "list", "--json"}),
	)
}
