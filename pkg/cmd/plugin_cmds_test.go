package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"

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

func TestFlagsArePassedAsArgs(t *testing.T) {
	Execute(context.Background())

	pluginCmd := createPluginCmd()
	rootCmd.AddCommand(pluginCmd.cmd)
	c, _, _ := executeCommandC(rootCmd, "test", "testarg", "--testflag")
	fmt.Println(c.Args)

	require.Equal(t, len(pluginCmd.ParsedArgs), 2)
	require.Equal(t, strings.Join(pluginCmd.ParsedArgs, " "), "testarg --testflag")
}
