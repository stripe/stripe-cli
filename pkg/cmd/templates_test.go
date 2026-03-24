package cmd

import (
	"os"
	"path/filepath"
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
	t.Setenv("CLAUDECODE", "1")
	assert.True(t, isAIAgent())
}

func TestIsAIAgent_NotDetected(t *testing.T) {
	// Ensure none of the agent env vars are set
	for _, key := range []string{"ANTIGRAVITY_CLI_ALIAS", "CLAUDECODE", "CLINE_ACTIVE", "CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI", "CURSOR_AGENT", "GEMINI_CLI", "OPENCODE"} {
		t.Setenv(key, "")
	}
	assert.False(t, isAIAgent())
}

func TestFormatAgentGuidance(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	output := formatAgentGuidance(cmd)
	require.NotEmpty(t, output)
	assert.Contains(t, output, "[Agent guidance]")
	assert.Contains(t, output, "--api-key")
	assert.Contains(t, output, "STRIPE_API_KEY")
	assert.Contains(t, output, "stripe resources")
	assert.Contains(t, output, "--stripe-account")
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
	for _, key := range []string{"ANTIGRAVITY_CLI_ALIAS", "CLAUDECODE", "CLINE_ACTIVE", "CODEX_SANDBOX", "CODEX_THREAD_ID", "CODEX_SANDBOX_NETWORK_DISABLED", "CODEX_CI", "CURSOR_AGENT", "GEMINI_CLI", "OPENCODE"} {
		t.Setenv(key, "")
	}

	root := &cobra.Command{Use: "stripe"}
	child := &cobra.Command{Use: "listen"}
	root.AddCommand(child)

	assert.Empty(t, aiAgentHelpTop(root))
	assert.Empty(t, aiAgentHelp(child))
}
