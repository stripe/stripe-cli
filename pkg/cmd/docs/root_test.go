package docs_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cliconfig "github.com/stripe/stripe-cli/pkg/config"

	"github.com/stripe/stripe-cli/internal/docs"
	"github.com/stripe/stripe-cli/internal/markdown"
	cmd "github.com/stripe/stripe-cli/pkg/cmd/docs"
)

func TestNew(t *testing.T) {
	root := cmd.New().Root()

	assert.Equal(t, "docs <path>", root.Use)
	assert.NotEmpty(t, root.Short)
}

func TestRootPrefixesPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/connect/accounts", r.URL.Path)
		fmt.Fprint(w, "# Accounts\n\nManage connected accounts.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"connect", "accounts"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Accounts")
}

func TestFetchPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept payments with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
}

// TestAgentDetectionDisablesTUI verifies that when an agent env var is set,
// running with no arguments prints help rather than launching the interactive
// TUI. In a real terminal (where term.IsTerminal is true) this would otherwise
// open BubbleTea; the test environment uses a bytes.Buffer so it also confirms
// no panic or hang occurs through the non-TUI code path.
func TestAgentDetectionDisablesTUI(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	var out bytes.Buffer
	root := cmd.New().Root()
	root.SetOut(&out)
	root.SetArgs([]string{})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Usage:", "expected help output, not TUI")
}

func TestAgentDetectionForcesNottyStyle(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.NotContains(t, result, "\x1b[", "should not contain ANSI escape codes when agent is detected")
}

func TestColorOffForcesNottyStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	cfg := &cliconfig.Config{Color: "off"}

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(cfg),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.NotContains(t, result, "\x1b[", "should not contain ANSI escape codes when --color=off")
}

func TestColorOnForcesColorEvenWithAgent(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept **payments** with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	cfg := &cliconfig.Config{Color: "on"}

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithConfig(cfg),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/payments"})

	err := root.ExecuteContext(context.Background())
	require.NoError(t, err)

	result := out.String()
	assert.Contains(t, result, "Payments")
	assert.Contains(t, result, "\x1b[", "should contain ANSI escape codes when --color=on even with agent")
}

func TestPreRun_LoggerRespectsConfiguredLevel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "# Test\n\nContent.")
	}))
	defer server.Close()

	var logBuf bytes.Buffer
	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
		cmd.WithLogger(logger),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"/test"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, logBuf.String(), "injected debug-level logger should capture log output")
}

func TestRootCommand_NoTUI_RendersOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "# Payments\n\nAccept payments with Stripe.")
	}))
	defer server.Close()

	client := docs.NewClient("test").WithOptions(docs.WithBaseURL(server.URL))
	renderer, err := markdown.NewRenderer()
	require.NoError(t, err)

	var out bytes.Buffer
	root := cmd.New().WithOptions(
		cmd.WithClient(client),
		cmd.WithRenderer(renderer),
	).Root()
	root.SetOut(&out)
	root.SetArgs([]string{"--non-interactive", "/payments"})

	err = root.ExecuteContext(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Payments")
}
