package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-cli/pkg/cmd/resource"
	"github.com/stripe/stripe-cli/pkg/cmd/resources"
	"github.com/stripe/stripe-cli/pkg/config"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/plugins"
)

const (
	pluginManifestURL = "https://stripe.jfrog.io/artifactory/stripe-cli-plugins-local/plugins.toml"
)

func newResourcesTestRoot(t *testing.T) *cobra.Command {
	t.Helper()

	cfg := &config.Config{}
	root := &cobra.Command{
		Use:         "stripe",
		Annotations: map[string]string{"resources": "resources"},
	}
	root.PersistentFlags().StringVar(&cfg.Profile.APIKey, "api-key", "", "Your API key to use for the command")
	root.AddCommand(newResourcesCmd().cmd)
	resources.AddAllResourcesCmds(root, cfg)
	require.NoError(t, resource.AddDatabasesCmd(root, cfg))
	require.NoError(t, resource.PostProcessResourceCommands(root, cfg))

	return root
}

func TestResources(t *testing.T) {
	output, err := executeCommand(newResourcesTestRoot(t), "resources")

	require.Contains(t, output, "Available commands:")
	require.NoError(t, err)
}

func TestResourcesHidesDatabases(t *testing.T) {
	output, err := executeCommand(newResourcesTestRoot(t), "resources")
	require.NoError(t, err)
	require.NotContains(t, output, "databases")
}

func TestResourcesListAliasedName(t *testing.T) {
	output, err := executeCommand(newResourcesTestRoot(t), "resources")
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

	root := newResourcesTestRoot(t)
	apiBase := fmt.Sprintf("--api-base=%s", ts.URL)
	apiKey := "--api-key=rk_test_1234567890"

	_, err := executeCommand(root, apiBase, apiKey, "invoice_line_items", "list", "in_123")
	require.NoError(t, err)
	_, err = executeCommand(root, apiBase, apiKey, "line_items", "list", "in_123")
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

	for _, cmd := range newResourcesTestRoot(t).Commands() {
		for _, pluginCommand := range pluginCommands {
			// TO-DO: this is a patch.
			// this check and this patch PR https://github.com/stripe/stripe-cli/pull/887
			// should be removed once openapi spec is updated to not use `apps`
			if cmd.Use == "apps" {
				continue
			}

			// TO-DO: This test fails if you have the "projects" plugin installed
			// because it looks at your real plugin list.
			if cmd.Use == "projects" {
				continue
			}

			require.False(t, cmd.Use == pluginCommand)
		}
	}
}
