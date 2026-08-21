package pluginalias

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFor_UnknownPluginHasNoAliases(t *testing.T) {
	spec := For("not-a-plugin")

	assert.Empty(t, spec.Names)
	assert.Empty(t, spec.ArgPrefix)
}

func TestFor_DirectoryPrefixesOnlyTheSubcommandAlias(t *testing.T) {
	spec := For("directory")

	require.Contains(t, spec.Names, "search")
	assert.Equal(t, map[string][]string{"search": {"search"}}, spec.ArgPrefix)
	// Every prefixed alias has to be one the command actually answers to, or the
	// prefix can never fire.
	for alias := range spec.ArgPrefix {
		assert.Contains(t, spec.Names, alias)
	}
}

// execute resolves argv through a root command so Cobra records CalledAs the way it
// does during a real invocation, then reports the args the plugin would receive.
func execute(t *testing.T, pluginName string, argv []string) []string {
	t.Helper()

	var got []string
	cmd := &cobra.Command{
		Use:                pluginName,
		Aliases:            For(pluginName).Names,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(cmd *cobra.Command, _ []string) error {
			got = PluginArgs(argv, cmd, pluginName)
			return nil
		},
	}

	root := &cobra.Command{Use: "stripe", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().String("log-level", "", "")
	root.AddCommand(cmd)
	root.SetArgs(argv[1:])

	require.NoError(t, root.Execute())

	return got
}

func TestPluginArgs(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "canonical name",
			argv: []string{"stripe", "directory", "search", "coffee shops"},
			want: []string{"search", "coffee shops"},
		},
		{
			name: "host flags before the plugin name are not forwarded",
			argv: []string{"stripe", "--log-level", "debug", "directory", "search", "x"},
			want: []string{"search", "x"},
		},
		{
			name: "plugin flags after the plugin name are forwarded",
			argv: []string{"stripe", "directory", "search", "x", "--limit", "5"},
			want: []string{"search", "x", "--limit", "5"},
		},
		{
			name: "typo alias forwards the remaining args untouched",
			argv: []string{"stripe", "direcotry", "search", "x"}, //nolint:misspell // Intentional typo alias.
			want: []string{"search", "x"},
		},
		{
			name: "subcommand alias restores the subcommand it stands in for",
			argv: []string{"stripe", "search", "coffee shops"},
			want: []string{"search", "coffee shops"},
		},
		{
			name: "bare invocation forwards no args",
			argv: []string{"stripe", "directory"},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, execute(t, "directory", tt.argv))
		})
	}
}

func TestPluginArgs_PluginWithoutAliases(t *testing.T) {
	assert.Equal(t, []string{"create", "x"}, execute(t, "apps", []string{"stripe", "apps", "create", "x"}))
}

// TestPluginArgs_WithoutCalledAs covers `stripe help <cmd>` and a plugin root whose
// subcommand stub ran instead: Cobra records CalledAs on neither, so the name the
// user typed has to be recovered from argv.
func TestPluginArgs_WithoutCalledAs(t *testing.T) {
	tests := []struct {
		name string
		argv []string
		want []string
	}{
		{
			name: "help subcommand on the plugin name",
			argv: []string{"stripe", "help", "directory"},
			want: []string{},
		},
		{
			name: "help subcommand on the plugin name with a subcommand",
			argv: []string{"stripe", "help", "directory", "search"},
			want: []string{"search"},
		},
		{
			name: "help subcommand on a subcommand alias",
			argv: []string{"stripe", "help", "search"},
			want: []string{"search"},
		},
		{
			name: "earliest name-or-alias wins so the plugin name is preferred",
			argv: []string{"stripe", "directory", "search", "x"},
			want: []string{"search", "x"},
		},
		{
			name: "falls back to the plugin name when nothing matches",
			argv: []string{"stripe", "listen"},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A command Cobra never executed reports an empty CalledAs.
			cmd := &cobra.Command{Use: "directory", Aliases: For("directory").Names}

			assert.Equal(t, tt.want, PluginArgs(tt.argv, cmd, "directory"))
		})
	}
}

func TestPluginArgs_NilCommand(t *testing.T) {
	assert.Equal(t, []string{"search", "x"}, PluginArgs([]string{"stripe", "directory", "search", "x"}, nil, "directory"))
}

func TestPluginArgs_DoesNotAliasInput(t *testing.T) {
	argv := []string{"stripe", "directory", "search"}

	got := PluginArgs(argv, nil, "directory")
	got[0] = "mutated"

	assert.Equal(t, []string{"stripe", "directory", "search"}, argv)
}
