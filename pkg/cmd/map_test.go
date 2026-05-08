package cmd

import (
	"bytes"
	"encoding/json"
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

// buildTestTree returns a root command with a small subtree for mode tests.
func buildTestTree() *cobra.Command {
	root := newTestCommand("stripe", "CLI root")
	listen := newTestCommand("listen", "Listen for webhook events")
	issuing := newTestCommand("issuing", "Issuing commands")
	cards := newTestCommand("cards", "Make requests on cards")
	create := newTestCommand("create", "Create a card")
	list := newTestCommand("list", "List all cards")
	cards.AddCommand(create, list)
	disputes := newTestCommand("disputes", "Make requests on disputes")
	retrieve := newTestCommand("retrieve", "Retrieve a dispute")
	disputes.AddCommand(retrieve)
	issuing.AddCommand(cards, disputes)
	trigger := newTestCommand("trigger", "Trigger a test webhook event")
	root.AddCommand(listen, issuing, trigger)
	return root
}

func TestMapHiddenCommandsExcluded(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	visible := newTestCommand("visible", "A visible command")
	hidden := newTestCommand("hidden", "A hidden command")
	hidden.Hidden = true

	root.AddCommand(visible, hidden)

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, root, mapModeTree)
	output := buf.String()

	assert.Contains(t, output, "child")
	assert.NotContains(t, output, "help")
}

func TestMapNestedTreeStructure(t *testing.T) {
	root := buildTestTree()

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, parent, mapModeTree)
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
	printCommandMap(&buf, leaf, mapModeTree)
	output := buf.String()

	// Leaf command has no children — output is just the command path.
	assert.Equal(t, "stripe version\n", output)
}

func TestMapDescriptionsShown(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	child := newTestCommand("listen", "Listen for webhooks")
	root.AddCommand(child)

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeTree)
	output := buf.String()

	assert.Contains(t, output, "listen")
	assert.Contains(t, output, "Listen for webhooks")
}

func TestMapFlagExistsOnRootCmd(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("map")
	require.NotNil(t, flag, "--map flag should be registered on rootCmd")
	assert.Equal(t, "", flag.DefValue)
	assert.Equal(t, "string", flag.Value.Type())
	assert.Equal(t, "tree", flag.NoOptDefVal)
}

func TestMapBoxDrawingCharacters(t *testing.T) {
	root := newTestCommand("app", "")
	a := newTestCommand("alpha", "First")
	b := newTestCommand("beta", "Second")
	c := newTestCommand("gamma", "Third")
	root.AddCommand(a, b, c)

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeTree)
	output := buf.String()

	// First and middle children use ├──, last child uses └──
	assert.Contains(t, output, "├── alpha")
	assert.Contains(t, output, "├── beta")
	assert.Contains(t, output, "└── gamma")
}

func TestGetMapMode(t *testing.T) {
	// Bare --map
	assert.Equal(t, mapModeTree, getMapMode([]string{"--map"}))
	assert.Equal(t, mapModeTree, getMapMode([]string{"issuing", "--map"}))
	assert.Equal(t, mapModeTree, getMapMode([]string{"--map", "issuing"}))

	// Explicit modes
	assert.Equal(t, mapModeTree, getMapMode([]string{"--map=tree"}))
	assert.Equal(t, mapModeCompact, getMapMode([]string{"--map=compact"}))
	assert.Equal(t, mapModePaths, getMapMode([]string{"--map=paths"}))
	assert.Equal(t, mapModeJSON, getMapMode([]string{"--map=json"}))

	// No flag
	assert.Equal(t, mapModeDefault, getMapMode([]string{"issuing"}))
	assert.Equal(t, mapModeDefault, getMapMode([]string{}))

	// "--map" after "--" should be ignored (positional arg, not flag)
	assert.Equal(t, mapModeDefault, getMapMode([]string{"--", "--map"}))

	// flags containing "map" as substring should not match
	assert.Equal(t, mapModeDefault, getMapMode([]string{"--mapfile"}))
	assert.Equal(t, mapModeDefault, getMapMode([]string{"--roadmap"}))

	// --map mixed with other flags
	assert.Equal(t, mapModeTree, getMapMode([]string{"-v", "--map", "-h"}))
	assert.Equal(t, mapModeCompact, getMapMode([]string{"-v", "--map=compact", "-h"}))

	// Invalid mode prints error to stderr
	var errBuf bytes.Buffer
	stderrOverride = &errBuf
	defer func() { stderrOverride = nil }()
	assert.Equal(t, mapModeDefault, getMapMode([]string{"--map=invalid"}))
	assert.Contains(t, errBuf.String(), "Unknown --map mode")
	assert.Contains(t, errBuf.String(), "invalid")
}

func TestStripMapFlag(t *testing.T) {
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"issuing", "--map"}))
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map", "issuing"}))
	assert.Equal(t, []string{}, stripMapFlag([]string{"--map"}))
	assert.Equal(t, []string{"a", "b"}, stripMapFlag([]string{"a", "b"}))
	// --map=value forms are stripped
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map=compact", "issuing"}))
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map=paths", "issuing"}))
	assert.Equal(t, []string{"issuing"}, stripMapFlag([]string{"--map=json", "issuing"}))
	// flags containing "map" as substring are NOT stripped
	assert.Equal(t, []string{"--mapfile"}, stripMapFlag([]string{"--mapfile"}))
}

func TestMapCompactMode(t *testing.T) {
	root := buildTestTree()

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeCompact)
	output := buf.String()

	// Should contain command names
	assert.Contains(t, output, "stripe")
	assert.Contains(t, output, "├── listen\n")
	assert.Contains(t, output, "├── cards\n")
	assert.Contains(t, output, "│   ├── create\n")
	assert.Contains(t, output, "│   └── list\n")
	assert.Contains(t, output, "└── trigger\n")

	// Should NOT contain descriptions
	assert.NotContains(t, output, "Listen for webhook events")
	assert.NotContains(t, output, "Create a card")
	assert.NotContains(t, output, "Retrieve a dispute")
}

func TestMapPathsMode(t *testing.T) {
	root := buildTestTree()

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModePaths)
	output := buf.String()

	// Should be flat list of full command paths, one per line
	assert.Contains(t, output, "stripe listen\n")
	assert.Contains(t, output, "stripe issuing cards create\n")
	assert.Contains(t, output, "stripe issuing cards list\n")
	assert.Contains(t, output, "stripe issuing disputes retrieve\n")
	assert.Contains(t, output, "stripe trigger\n")

	// Should NOT contain tree-drawing characters
	assert.NotContains(t, output, "├──")
	assert.NotContains(t, output, "└──")
	assert.NotContains(t, output, "│")
}

func TestMapJSONMode(t *testing.T) {
	root := buildTestTree()

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeJSON)
	output := buf.String()

	// Should be valid JSON
	var node commandNode
	err := json.Unmarshal(buf.Bytes(), &node)
	require.NoError(t, err, "JSON output should be valid: %s", output)

	// Root node
	assert.Equal(t, "stripe", node.Name)
	assert.Equal(t, "CLI root", node.Desc)
	require.Len(t, node.Commands, 3) // listen, issuing, trigger (cobra sorts alphabetically)

	// Find issuing node
	var issuing *commandNode
	for i := range node.Commands {
		if node.Commands[i].Name == "issuing" {
			issuing = &node.Commands[i]
			break
		}
	}
	require.NotNil(t, issuing)
	require.Len(t, issuing.Commands, 2) // cards, disputes

	// Find cards under issuing
	var cards *commandNode
	for i := range issuing.Commands {
		if issuing.Commands[i].Name == "cards" {
			cards = &issuing.Commands[i]
			break
		}
	}
	require.NotNil(t, cards)
	require.Len(t, cards.Commands, 2)
	assert.Equal(t, "create", cards.Commands[0].Name)
	assert.Equal(t, "Create a card", cards.Commands[0].Desc)
}

func TestMapJSONLeafHasNoCommands(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	leaf := newTestCommand("version", "Print version")
	root.AddCommand(leaf)

	var buf bytes.Buffer
	printCommandMap(&buf, leaf, mapModeJSON)

	var node commandNode
	err := json.Unmarshal(buf.Bytes(), &node)
	require.NoError(t, err)
	assert.Equal(t, "version", node.Name)
	assert.Nil(t, node.Commands, "leaf command should have no commands array")
}

func TestMapDeprecatedCommandsExcluded(t *testing.T) {
	root := newTestCommand("stripe", "CLI root")
	active := newTestCommand("active", "An active command")
	deprecated := newTestCommand("old", "A deprecated command")
	deprecated.Deprecated = "use 'active' instead"
	root.AddCommand(active, deprecated)

	var buf bytes.Buffer
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, root, mapModeTree)
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
	printCommandMap(&buf, root, mapModeTree)
	output := buf.String()

	assert.Contains(t, output, "myplugin")
	assert.Contains(t, output, "A simple plugin")
}

func TestParseMapMode(t *testing.T) {
	mode, ok := parseMapMode("--map")
	assert.Equal(t, mapModeTree, mode)
	assert.True(t, ok)

	mode, ok = parseMapMode("--map=tree")
	assert.Equal(t, mapModeTree, mode)
	assert.True(t, ok)

	mode, ok = parseMapMode("--map=compact")
	assert.Equal(t, mapModeCompact, mode)
	assert.True(t, ok)

	mode, ok = parseMapMode("--map=paths")
	assert.Equal(t, mapModePaths, mode)
	assert.True(t, ok)

	mode, ok = parseMapMode("--map=json")
	assert.Equal(t, mapModeJSON, mode)
	assert.True(t, ok)

	mode, ok = parseMapMode("--map=invalid")
	assert.Equal(t, mapModeDefault, mode)
	assert.False(t, ok)

	// Non-map args
	mode, ok = parseMapMode("--verbose")
	assert.Equal(t, mapModeDefault, mode)
	assert.False(t, ok)
}
