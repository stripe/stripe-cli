package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

func TestGetLogin(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := config.Config{}
	expected := `
Before using the CLI, you'll need to login:

  $ stripe login

If you're working on multiple projects, you can run the login command with the
--project-name flag:

  $ stripe login --project-name rocket-rides`
	output := getLogin(&fs, &cfg)

	assert.Equal(t, expected, output)
}

func TestGetLoginEmpty(t *testing.T) {
	fs := afero.NewMemMapFs()
	cfg := config.Config{}

	file := filepath.Join(cfg.GetConfigFolder(os.Getenv("XDG_CONFIG")), "config.toml")

	afero.WriteFile(fs, file, []byte{}, os.ModePerm)

	output := getLogin(&fs, &cfg)

	assert.Equal(t, "", output)
}

func TestIsAIAgent_Detected(t *testing.T) {
	t.Setenv("AI_AGENT", "")
	t.Setenv("CLAUDECODE", "1")
	assert.True(t, isAIAgent())
}

func TestIsAIAgent_NotDetected(t *testing.T) {
	// Ensure none of the agent env vars are set
	for _, key := range []string{
		"AI_AGENT",
		"ANTIGRAVITY_CLI_ALIAS", "CLAUDECODE", "CLAUDE_CODE", "CLINE_ACTIVE",
		"CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI",
		"CURSOR_TRACE_ID", "CURSOR_AGENT", "GEMINI_CLI", "OPENCODE", "OPENCODE_CLIENT",
		"OPENCLAW_SHELL",
	} {
		t.Setenv(key, "")
	}
	assert.False(t, isAIAgent())
}

func TestFormatAgentGuidance(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	output := formatAgentGuidance(cmd)
	require.NotEmpty(t, output)
	assert.Contains(t, output, "[Agent guidance]")
	assert.Contains(t, output, "stripe sandbox create")
	assert.Contains(t, output, "--api-key")
	assert.Contains(t, output, "STRIPE_API_KEY")
	assert.Contains(t, output, "stripe --map")
	assert.Contains(t, output, "stripe resources")
	assert.NotContains(t, output, "--stripe-account", "should not show --stripe-account when flag is not defined")
	assert.NotContains(t, output, "-d", "should not show -d when data flag is not defined")
}

func TestFormatAgentGuidance_WithAnnotation(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Custom tip for this command.",
		},
	}
	output := formatAgentGuidance(cmd)
	require.NotEmpty(t, output)
	assert.Contains(t, output, "[Agent guidance]")
	assert.Contains(t, output, "Custom tip for this command.")
}

func TestFormatAgentGuidance_AnnotationRendersBeforeSharedTips(t *testing.T) {
	cmd := &cobra.Command{
		Use: "test",
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Per-command tip.",
		},
	}
	output := formatAgentGuidance(cmd)
	annotationIdx := strings.Index(output, "Per-command tip")
	apiKeyIdx := strings.Index(output, "--api-key")
	require.Greater(t, annotationIdx, 0, "annotation should be present")
	require.Greater(t, apiKeyIdx, 0, "shared tip should be present")
	assert.Less(t, annotationIdx, apiKeyIdx, "per-command annotation should render before shared tips")
}

func TestFormatAgentGuidance_DataFlagShownWhenPresent(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().StringArrayP("data", "d", nil, "Data for the API request")

	output := formatAgentGuidance(cmd)
	assert.Contains(t, output, "-d", "should show -d tip when data flag is defined")
}

func TestFormatAgentGuidance_StripeAccountShownWhenPresent(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("stripe-account", "", "Set a header identifying the connected account")

	output := formatAgentGuidance(cmd)
	assert.Contains(t, output, "--stripe-account", "should show --stripe-account when flag is defined")
}

func TestAIAgentHelpTop_RootOnly(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	root := &cobra.Command{Use: "stripe"}
	child := &cobra.Command{Use: "listen"}
	root.AddCommand(child)

	assert.NotEmpty(t, aiAgentHelpTop(root), "should render for root command")
	assert.Empty(t, aiAgentHelpTop(child), "should not render for subcommand")
}

func TestAIAgentHelp_SubcommandOnly(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	root := &cobra.Command{Use: "stripe"}
	child := &cobra.Command{Use: "listen"}
	root.AddCommand(child)

	assert.Empty(t, aiAgentHelp(root), "should not render for root command")
	assert.NotEmpty(t, aiAgentHelp(child), "should render for subcommand")
}

func TestAIAgentHelp_NotDetected(t *testing.T) {
	for _, key := range []string{
		"AI_AGENT",
		"ANTIGRAVITY_CLI_ALIAS", "CLAUDECODE", "CLAUDE_CODE", "CLINE_ACTIVE",
		"CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI",
		"CURSOR_TRACE_ID", "CURSOR_AGENT", "GEMINI_CLI", "OPENCODE", "OPENCODE_CLIENT",
		"OPENCLAW_SHELL",
	} {
		t.Setenv(key, "")
	}

	root := &cobra.Command{Use: "stripe"}
	child := &cobra.Command{Use: "listen"}
	root.AddCommand(child)

	assert.Empty(t, aiAgentHelpTop(root))
	assert.Empty(t, aiAgentHelp(child))
}

func TestWrappedRequestParamsFlagUsages_FormatAnnotation(t *testing.T) {
	cmd := &cobra.Command{Use: "create", Annotations: make(map[string]string)}

	// String param with format: should show the format label instead of "string".
	cmd.Flags().String("created", "", "")
	cmd.Flags().SetAnnotation("created", "request", []string{"true"})
	cmd.Flags().SetAnnotation("created", "apitype", []string{"integer"})
	cmd.Flags().SetAnnotation("created", "format", []string{"unix-time"})

	// String param without format: should show the raw type label.
	cmd.Flags().String("description", "", "")
	cmd.Flags().SetAnnotation("description", "request", []string{"true"})
	cmd.Flags().SetAnnotation("description", "apitype", []string{"string"})

	output := WrappedRequestParamsFlagUsages(cmd)
	assert.Contains(t, output, "--created <unix-time>")
	assert.Contains(t, output, "--description <string>")
}

// noop is a minimal RunE so Cobra considers the command "available".
var noop = func(cmd *cobra.Command, args []string) error { return nil }

// Test that subcommands with ai_agent_help annotations don't render root
// command groups (Webhook commands, Resource commands, etc.). This was a
// bug where .Annotations was used as a root-vs-subcommand signal.
func TestUsageTemplate_SubcommandWithAnnotationDoesNotShowRootGroups(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":    "http",
			"listen": "webhooks",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	child := &cobra.Command{
		Use:   "login",
		Short: "Login to your Stripe account",
		RunE:  noop,
		Annotations: map[string]string{
			AIAgentHelpAnnotationKey: "  Use --interactive for non-browser auth.",
		},
	}
	root.AddCommand(child)

	output := child.UsageString()
	assert.NotContains(t, output, "Webhook commands", "subcommand should not render root command groups")
	assert.NotContains(t, output, "Resource commands", "subcommand should not render root command groups")
	assert.Contains(t, output, "login", "subcommand should show its own usage")
}

// Test that the root command still renders its categorized command groups.
func TestUsageTemplate_RootShowsCommandGroups(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":    "http",
			"listen": "webhooks",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	listen := &cobra.Command{Use: "listen", Short: "Listen for webhook events", RunE: noop}
	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	root.AddCommand(listen, get)

	output := root.UsageString()
	assert.Contains(t, output, "Webhook commands")
	assert.Contains(t, output, "API commands")
}

func TestHasAnnotatedCommands(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"tools": "installed_plugin",
			"apps":  "available_plugin",
		},
	}
	root.AddCommand(&cobra.Command{Use: "tools", RunE: noop})
	root.AddCommand(&cobra.Command{Use: "apps", RunE: noop})
	root.AddCommand(&cobra.Command{Use: "login", RunE: noop})

	assert.True(t, hasAnnotatedCommands(root, "installed_plugin"))
	assert.True(t, hasAnnotatedCommands(root, "available_plugin"))
	assert.False(t, hasAnnotatedCommands(root, "nonexistent"))
}

func TestHasAnnotatedCommands_NoMatches(t *testing.T) {
	root := &cobra.Command{
		Use:         "stripe",
		Annotations: map[string]string{},
	}
	root.AddCommand(&cobra.Command{Use: "login", RunE: noop})

	assert.False(t, hasAnnotatedCommands(root, "installed_plugin"))
}

func TestPluginDescription_UsesLongWhenPresent(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Short text",
		Long:  "A longer description of the plugin.",
	}
	// Attach to a parent so NamePadding() is valid.
	parent := &cobra.Command{Use: "stripe"}
	parent.AddCommand(cmd)

	result := pluginDescription(cmd)
	assert.Contains(t, result, "A longer description of the plugin.")
	assert.NotContains(t, result, "Short text")
}

func TestPluginDescription_FallsBackToShort(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Fallback short description",
	}
	parent := &cobra.Command{Use: "stripe"}
	parent.AddCommand(cmd)

	result := pluginDescription(cmd)
	assert.Contains(t, result, "Fallback short description")
}

func TestPluginDescription_WrapsLongText(t *testing.T) {
	longText := "Manage your account from the terminal and update settings like branding, checkout color and more. These capabilities are typically accessible only from the dashboard and not available on the public API."
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Short",
		Long:  longText,
	}
	parent := &cobra.Command{Use: "stripe"}
	parent.AddCommand(cmd)

	result := pluginDescription(cmd)
	lines := strings.Split(result, "\n")
	assert.Greater(t, len(lines), 1, "long text should wrap to multiple lines")
	for _, line := range lines[1:] {
		assert.True(t, strings.HasPrefix(line, "    "), "continuation lines should be indented")
	}
}

func TestUsageTemplate_InstalledPluginsSection(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":   "http",
			"tools": "installed_plugin",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	tools := &cobra.Command{Use: "tools", Short: "Manage account", Long: "Manage your account from the terminal.", RunE: noop}
	root.AddCommand(get, tools)

	output := root.UsageString()
	assert.Contains(t, output, "Installed plugins:")
	assert.Contains(t, output, "tools")
	assert.Contains(t, output, "Manage your account from the terminal.")
	assert.NotContains(t, output, "Available plugins:")
}

func TestUsageTemplate_AvailablePluginsSection(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":  "http",
			"apps": "available_plugin",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	apps := &cobra.Command{Use: "apps", Short: "Build and manage Stripe Apps.", RunE: noop}
	root.AddCommand(get, apps)

	output := root.UsageString()
	assert.Contains(t, output, "Available plugins:")
	assert.Contains(t, output, "apps")
	assert.Contains(t, output, "Build and manage Stripe Apps.")
	assert.NotContains(t, output, "Installed plugins:")
}

func TestUsageTemplate_BothPluginSections(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":   "http",
			"tools": "installed_plugin",
			"apps":  "available_plugin",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	tools := &cobra.Command{Use: "tools", Short: "Manage account", Long: "Manage your account from the terminal.", RunE: noop}
	apps := &cobra.Command{Use: "apps", Short: "Build and manage Stripe Apps.", RunE: noop}
	root.AddCommand(get, tools, apps)

	output := root.UsageString()
	assert.Contains(t, output, "Installed plugins:")
	assert.Contains(t, output, "Available plugins:")
	installedIdx := strings.Index(output, "Installed plugins:")
	availableIdx := strings.Index(output, "Available plugins:")
	assert.Less(t, installedIdx, availableIdx, "Installed plugins should appear before Available plugins")
}

func TestUsageTemplate_PluginsExcludedFromOtherCommands(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":   "http",
			"tools": "installed_plugin",
			"apps":  "available_plugin",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	tools := &cobra.Command{Use: "tools", Short: "Manage account", RunE: noop}
	apps := &cobra.Command{Use: "apps", Short: "Build Stripe Apps", RunE: noop}
	login := &cobra.Command{Use: "login", Short: "Login to Stripe", RunE: noop}
	root.AddCommand(get, tools, apps, login)

	output := root.UsageString()
	otherIdx := strings.Index(output, "Other commands:")
	installedIdx := strings.Index(output, "Installed plugins:")

	// "login" should be in Other commands, not in plugin sections
	loginIdx := strings.Index(output, "login")
	assert.Greater(t, loginIdx, otherIdx, "login should appear after Other commands header")
	assert.Less(t, loginIdx, installedIdx, "login should appear before Installed plugins header")
}

func TestUsageTemplate_InstalledPluginFallsBackToShort(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get":    "http",
			"simple": "installed_plugin",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	simple := &cobra.Command{Use: "simple", Short: "A simple plugin with no Long", RunE: noop}
	root.AddCommand(get, simple)

	output := root.UsageString()
	assert.Contains(t, output, "Installed plugins:")
	assert.Contains(t, output, "A simple plugin with no Long")
}

func TestUsageTemplate_NoPluginSectionsWhenNoPlugins(t *testing.T) {
	root := &cobra.Command{
		Use: "stripe",
		Annotations: map[string]string{
			"get": "http",
		},
	}
	root.SetUsageTemplate(getUsageTemplate())

	get := &cobra.Command{Use: "get", Short: "Make GET requests", RunE: noop}
	login := &cobra.Command{Use: "login", Short: "Login", RunE: noop}
	root.AddCommand(get, login)

	output := root.UsageString()
	assert.NotContains(t, output, "Installed plugins:")
	assert.NotContains(t, output, "Available plugins:")
}
