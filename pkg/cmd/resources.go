package cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pelletier/go-toml"
	"github.com/spf13/cobra"

	"github.com/stripe/stripe-cli/pkg/ansi"
	"github.com/stripe/stripe-cli/pkg/validators"
)

const (
	pluginManifestUrl = "https://stripe.jfrog.io/artifactory/stripe-cli-plugins-local/plugins.toml"
)

type resourcesCmd struct {
	cmd *cobra.Command
}

// PluginList contains a list of plugins
type PluginList struct {
	Plugins []Plugin `toml:"Plugin"`
}

// Plugin contains the plugin properties
type Plugin struct {
	Shortname        string
	Binary           string
	Releases         []Release `toml:"Release"`
	MagicCookieValue string
}

// Release is the type that holds release data for a specific build of a plugin
type Release struct {
	Arch    string
	OS      string
	Version string
	Sum     string
}

func newResourcesCmd() *resourcesCmd {
	rc := &resourcesCmd{}

	rc.cmd = &cobra.Command{
		Use:   "resources",
		Args:  validators.NoArgs,
		Short: "List resource commands",
	}
	rc.cmd.SetHelpTemplate(getResourcesHelpTemplate())

	return rc
}

func getResourcesHelpTemplate() string {
	// This template uses `.Parent` to access subcommands on the root command.
	return fmt.Sprintf(`%s{{range $index, $cmd := .Parent.Commands}}{{if (or (eq (index $.Parent.Annotations $cmd.Name) "resource") (eq (index $.Parent.Annotations $cmd.Name) "namespace"))}}
  {{rpad $cmd.Name $cmd.NamePadding }} {{$cmd.Short}}{{end}}{{end}}

Use "stripe [command] --help" for more information about a command.
`,
		ansi.Bold("Available commands:"),
	)
}

func getAllPluginCommands() ([]string, error) {
	var manifest PluginList

	resp, err := http.Get(pluginManifestUrl)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}

	err = toml.Unmarshal(respBytes, &manifest)
	if err != nil {
		return []string{}, err
	}

	var pluginCommands []string
	for _, plugin := range manifest.Plugins {
		pluginCommands = append(pluginCommands, plugin.Shortname)
	}

	return pluginCommands, nil
}
