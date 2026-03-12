package cmd

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/ansi"
)

func init() {
	// Disable ANSI colors in tests so assertions match plain text.
	ansi.DisableColors = true
}

// newTestCommand creates a cobra.Command with the given use string and short description.
func newTestCommand(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
	}
}

func TestMapHiddenCommandsExcluded(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	visible := newTestCommand("visible", "A visible command")
	hidden := newTestCommand("hidden", "A hidden command")
	hidden.Hidden = true

	root.AddCommand(visible, hidden)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "visible")
	assert.NotContains(t, output, "hidden")
}

func TestMapHelpCommandExcluded(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	child := newTestCommand("child", "A child command")
	root.AddCommand(child)

	// Cobra auto-generates a "help" command when Commands() is called.
	// InitDefaultHelpCmd makes it explicit.
	root.InitDefaultHelpCmd()

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "child")
	assert.NotContains(t, output, "help")
}

func TestMapNestedTreeStructure(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")

	issuing := newTestCommand("issuing", "Issuing commands")
	cards := newTestCommand("cards", "Make requests on cards")
	create := newTestCommand("create", "Create a card")
	list := newTestCommand("list", "List all cards")
	cards.AddCommand(create, list)

	disputes := newTestCommand("disputes", "Make requests on disputes")
	retrieve := newTestCommand("retrieve", "Retrieve a dispute")
	disputes.AddCommand(retrieve)

	issuing.AddCommand(cards, disputes)
	root.AddCommand(issuing)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "stripe")
	assert.Contains(t, output, "├── cards")
	assert.Contains(t, output, "│   ├── create")
	assert.Contains(t, output, "│   └── list")
	assert.Contains(t, output, "└── disputes")
	assert.Contains(t, output, "    └── retrieve")
}

func TestMapSubtreeScoping(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	parent := newTestCommand("issuing", "Issuing commands")
	child := newTestCommand("cards", "Card operations")
	parent.AddCommand(child)
	root.AddCommand(parent)

	var buf bytes.Buffer
	printCommandMap(&buf, parent)
	output := buf.String()

	// The root line should be the scoped command path.
	assert.Contains(t, output, "stripe issuing")
	assert.Contains(t, output, "cards")
	// Root-level "stripe" alone should not appear as a separate line.
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	assert.Contains(t, string(lines[0]), "stripe issuing")
}

func TestMapLeafCommand(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	leaf := newTestCommand("version", "Print version")
	root.AddCommand(leaf)

	var buf bytes.Buffer
	printCommandMap(&buf, leaf)
	output := buf.String()

	// Leaf command has no children — output is just the command path.
	assert.Equal(t, "stripe version\n", output)
}

func TestMapDescriptionsShown(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	child := newTestCommand("listen", "Listen for webhooks")
	root.AddCommand(child)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "listen")
	assert.Contains(t, output, "Listen for webhooks")
}

func TestMapFlagExistsOnRootCmd(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("map")
	require.NotNil(t, flag, "--map flag should be registered on rootCmd")
	assert.Equal(t, "false", flag.DefValue)
	assert.Equal(t, "bool", flag.Value.Type())
}

func TestMapBoxDrawingCharacters(t *testing.T) {
	root := newTestCommand("app", "")
	a := newTestCommand("alpha", "First")
	b := newTestCommand("beta", "Second")
	c := newTestCommand("gamma", "Third")
	root.AddCommand(a, b, c)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	// First and middle children use ├──, last child uses └──
	assert.Contains(t, output, "├── alpha")
	assert.Contains(t, output, "├── beta")
	assert.Contains(t, output, "└── gamma")
}

func TestHasMapFlag(t *testing.T) {
	assert.True(t, hasMapFlag([]string{"--map"}))
	assert.True(t, hasMapFlag([]string{"issuing", "--map"}))
	assert.True(t, hasMapFlag([]string{"--map", "issuing"}))
	assert.False(t, hasMapFlag([]string{"issuing"}))
	assert.False(t, hasMapFlag([]string{}))
	// "--map" after "--" should be ignored (positional arg, not flag)
	assert.False(t, hasMapFlag([]string{"--", "--map"}))
	// --map=value forms
	assert.True(t, hasMapFlag([]string{"--map=true"}))
	assert.False(t, hasMapFlag([]string{"--map=false"}))
	// flags containing "map" as substring should not match
	assert.False(t, hasMapFlag([]string{"--mapfile"}))
	assert.False(t, hasMapFlag([]string{"--roadmap"}))
	// --map mixed with other flags
	assert.True(t, hasMapFlag([]string{"-v", "--map", "-h"}))
}

func TestStripMapFlag(t *testing.T) {
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"issuing", "--map"}))
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map", "issuing"}))
	assert.Equal(t, []string{}, stripMapFlag([]string{"--map"}))
	assert.Equal(t, []string{"a", "b"}, stripMapFlag([]string{"a", "b"}))
	// --map=value forms are stripped
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"issuing", "--map=true"}))
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map=false", "issuing"}))
	// flags containing "map" as substring are NOT stripped
	assert.Equal(t, []string{"--mapfile"}, stripMapFlag([]string{"--mapfile"}))
}

func TestMapDeprecatedCommandsExcluded(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	active := newTestCommand("active", "An active command")
	deprecated := newTestCommand("old", "A deprecated command")
	deprecated.Deprecated = "use 'active' instead"
	root.AddCommand(active, deprecated)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "active")
	assert.NotContains(t, output, "old")
}

func TestMapDeeplyNestedTree(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	l1 := newTestCommand("billing", "Billing commands")
	l2a := newTestCommand("meters", "Meter commands")
	l2b := newTestCommand("alerts", "Alert commands")
	l3 := newTestCommand("events", "Meter events")
	l4 := newTestCommand("list", "List events")
	l3.AddCommand(l4)
	l2a.AddCommand(l3)
	l1.AddCommand(l2a, l2b)
	root.AddCommand(l1)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	// Verify 3+ level prefix accumulation (cobra sorts alphabetically)
	assert.Contains(t, output, "└── billing")
	assert.Contains(t, output, "    ├── alerts")
	assert.Contains(t, output, "    └── meters")
	assert.Contains(t, output, "        └── events")
	assert.Contains(t, output, "            └── list")
}

func TestMapCommandWithEmptyDescription(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	noDesc := newTestCommand("nodesc", "")
	withDesc := newTestCommand("withdesc", "Has a description")
	root.AddCommand(noDesc, withDesc)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	// Command without description should not have trailing spaces
	assert.Contains(t, output, "├── nodesc\n")
	assert.Contains(t, output, "└── withdesc  Has a description")
}

func TestMapPluginWithSubcommands(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")

	// Simulate a plugin with manifest-declared subcommands
	pluginCmd := newTestCommand("apps", "Manage Stripe apps")
	pluginCmd.Annotations = map[string]string{"scope": "plugin"}
	createCmd := newTestCommand("create", "Create a new app")
	logsCmd := newTestCommand("logs", "View app logs")
	tailCmd := newTestCommand("tail", "Tail logs in real-time")
	logsCmd.AddCommand(tailCmd)
	pluginCmd.AddCommand(createCmd, logsCmd)
	root.AddCommand(pluginCmd)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "└── apps")
	assert.Contains(t, output, "create")
	assert.Contains(t, output, "logs")
	assert.Contains(t, output, "tail")
	assert.Contains(t, output, "Tail logs in real-time")
}

func TestMapPluginWithoutSubcommands(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	pluginCmd := newTestCommand("myplugin", "A simple plugin")
	pluginCmd.Annotations = map[string]string{"scope": "plugin"}
	root.AddCommand(pluginCmd)

	var buf bytes.Buffer
	printCommandMap(&buf, root)
	output := buf.String()

	assert.Contains(t, output, "myplugin")
	assert.Contains(t, output, "A simple plugin")
}
