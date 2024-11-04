package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"

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

func TestResourcesListAliasedName(t *testing.T) {
	output, err := executeCommand(rootCmd, "resources")
	require.NoError(t, err)

	assert.Contains(t, output, "Available commands:")

	aliases := resource.GetAliases()
	for principle, alias := range aliases {
		aliasRegexp := fmt.Sprintf("\n\\s+%s(s?)\\s+\n", resource.GetResourceCmdName(alias))
		principleRegexp := fmt.Sprintf("\n\\s+%s(s?)\\s+\n", resource.GetResourceCmdName(principle))
		assert.Regexp(t, regexp.MustCompile(aliasRegexp), output)
		assert.NotRegexp(t, regexp.MustCompile(principleRegexp), output)
	}
}

func TestAliasedResourcesCallPrincipleAPI(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/v1/invoices/in_123/lines")
	}))
	defer ts.Close()

	apiBase := fmt.Sprintf("--api-base=%s", ts.URL)
	apiKey := "--api-key=rk_test_1234567890"

	_, err := executeCommand(rootCmd, apiBase, apiKey, "invoice_line_items", "list", "in_123")
	require.NoError(t, err)
	_, err = executeCommand(rootCmd, apiBase, apiKey, "line_items", "list", "in_123")
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
