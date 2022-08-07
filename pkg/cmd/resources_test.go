package cmd

import (
	"io"
	"net/http"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins"
)

const (
	pluginManifestURL = "https://stripe.jfrog.io/artifactory/stripe-cli-plugins-local/plugins.toml"
)

func TestResources(t *testing.T) {
	output, err := executeCommand(rootCmd, "resources")

	require.Contains(t, output, "Available commands:")
	require.NoError(t, err)
}

func TestConflictWithPluginCommand(t *testing.T) {
	// directly downloading the manifest can only be done within this unit test
	// plugins.GetPluginList should be used under normal circumstances
	resp, err := http.Get(pluginManifestURL)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var manifest plugins.PluginList
	err = toml.Unmarshal(respBytes, &manifest)
	require.NoError(t, err)

	var pluginCommands []string
	for _, plugin := range manifest.Plugins {
		pluginCommands = append(pluginCommands, plugin.Shortname)
	}

	for _, cmd := range rootCmd.Commands() {
		for _, pluginCommand := range pluginCommands {
			// TO-DO: this is a patch.
			// this check and this patch PR https://github.com/stripe/stripe-cli/pull/887
			// should be removed once openapi spec is updated to not use `apps`
			if cmd.Use == "apps" {
				continue
			}
			require.False(t, cmd.Use == pluginCommand)
		}
	}
}
