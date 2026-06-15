package pluginhints

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
)

// newTestCmd builds a pluginHintCmd with all side effects mocked out.
// By default accountIDFn reports a logged-in account; override in tests that
// need to simulate an unauthenticated user.
func newTestCmd(name string, opts ...option) *pluginHintCmd {
	p := &pluginHintCmd{
		name:        name,
		description: "Test description.",
		stdout:      &bytes.Buffer{},
		stdin:       strings.NewReader(""),
		accountIDFn: func() (string, error) { return "acct_test", nil },
		loginFn:     func(ctx context.Context) error { return nil },
	}
	for _, opt := range opts {
		opt(p)
	}
	p.Command = &cobra.Command{Use: name, RunE: p.run}
	return p
}

func (p *pluginHintCmd) output() string {
	return p.stdout.(*bytes.Buffer).String()
}

func findChildCommand(rootCmd *cobra.Command, name string) *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

// --- AddHintCommands ---

func TestAddHintCommands_DirectoryHintRoutesAliasesWhenPluginMissing(t *testing.T) {
	rootCmd := &cobra.Command{Use: "stripe"}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{})

	directoryCmd := findChildCommand(rootCmd, "directory")
	require.NotNil(t, directoryCmd)

	for _, name := range []string{
		"directory",
		"search",
		"directry",
		"directary",
		"direcotry", //nolint:misspell // Intentional typo alias.
		"diretory",
	} {
		t.Run(name, func(t *testing.T) {
			resolvedCmd, _, err := rootCmd.Find([]string{name})

			require.NoError(t, err)
			assert.Same(t, directoryCmd, resolvedCmd)
		})
	}
}

func TestAddHintCommands_DirectoryHintSkippedWhenPluginInstalled(t *testing.T) {
	rootCmd := &cobra.Command{Use: "stripe"}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{
		"directory": true,
	})

	assert.Nil(t, findChildCommand(rootCmd, "directory"))
}

// --- run ---

func TestRun_PluginFound_CallsPromptInstall(t *testing.T) {
	p := newTestCmd("generate", withPrivatePreview())
	installCalled := false
	p.lookupFn = func(ctx context.Context) error { return nil }
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.output(), "The \"generate\" plugin is required")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_LoggedIn_PrintsInstallHint(t *testing.T) {
	p := newTestCmd("apps") // accountIDFn returns "acct_test" by default
	p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Contains(t, p.output(), "stripe plugin install apps")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_NotLoggedIn_PromptsLogin(t *testing.T) {
	p := newTestCmd("docs")
	p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }
	p.accountIDFn = func() (string, error) { return "", nil }
	loginCalled := false
	p.loginFn = func(ctx context.Context) error { loginCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, loginCalled)
	assert.Contains(t, p.output(), "stripe login")
	assert.NotContains(t, p.output(), "stripe plugin install")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_AccountIDError_PromptsLogin(t *testing.T) {
	p := newTestCmd("docs")
	p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }
	p.accountIDFn = func() (string, error) { return "", errors.New("not configured") }
	loginCalled := false
	p.loginFn = func(ctx context.Context) error { loginCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, loginCalled)
	assert.Contains(t, p.output(), "stripe login")
	assert.NotContains(t, p.output(), "stripe plugin install")
}

func TestRun_PluginNotFound_PrivatePreviewTrue_ExitsWithOne(t *testing.T) {
	// Subprocess path: run the code that calls os.Exit(1).
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		p := &pluginHintCmd{
			name:           "generate",
			description:    "Test description.",
			privatePreview: true,
			stdout:         os.Stdout,
			stdin:          strings.NewReader(""),
		}
		p.Command = &cobra.Command{Use: "generate", RunE: p.run}
		p.lookupFn = func(ctx context.Context) error { return errors.New("not found") }
		p.accountIDFn = func() (string, error) { return "acct_123", nil }
		p.run(p.Command, nil) //nolint:errcheck
		return
	}

	var stdout bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=TestRun_PluginNotFound_PrivatePreviewTrue_ExitsWithOne")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	cmd.Stdout = &stdout

	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stdout.String(), "private preview")
	assert.Contains(t, stdout.String(), "acct_123")
}

// --- promptInstall ---

func TestPromptInstall_EnterKey_InstallsPlugin(t *testing.T) {
	p := newTestCmd("generate", withPrivatePreview())
	p.stdin = strings.NewReader("\n")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.promptInstall(context.Background())

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.output(), "installation complete")
}

func TestPromptInstall_OtherInput_CancelsInstall(t *testing.T) {
	p := newTestCmd("generate", withPrivatePreview())
	p.stdin = strings.NewReader("n\n")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.promptInstall(context.Background())

	assert.EqualError(t, err, "installation canceled")
	assert.False(t, installCalled)
}

func TestPromptInstall_InstallError_ReturnsError(t *testing.T) {
	p := newTestCmd("generate", withPrivatePreview())
	p.stdin = strings.NewReader("\n")
	p.installFn = func(ctx context.Context) error { return errors.New("install failed") }

	err := p.promptInstall(context.Background())

	assert.EqualError(t, err, "install failed")
}

// --- promptLogin ---

func TestPromptLogin_EnterKey_LogsIn(t *testing.T) {
	p := newTestCmd("docs")
	p.stdin = strings.NewReader("\n")
	loginCalled := false
	p.loginFn = func(ctx context.Context) error { loginCalled = true; return nil }

	err := p.promptLogin(context.Background())

	require.NoError(t, err)
	assert.True(t, loginCalled)
	assert.Contains(t, p.output(), "stripe login")
}

func TestPromptLogin_OtherInput_CancelsLogin(t *testing.T) {
	p := newTestCmd("docs")
	p.stdin = strings.NewReader("n\n")
	loginCalled := false
	p.loginFn = func(ctx context.Context) error { loginCalled = true; return nil }

	err := p.promptLogin(context.Background())

	assert.EqualError(t, err, "login canceled")
	assert.False(t, loginCalled)
}

func TestPromptLogin_LoginError_ReturnsError(t *testing.T) {
	p := newTestCmd("docs")
	p.stdin = strings.NewReader("\n")
	p.loginFn = func(ctx context.Context) error { return errors.New("login failed") }

	err := p.promptLogin(context.Background())

	assert.EqualError(t, err, "login failed")
}

// --- suggestNotAvailable ---

func TestSuggestNotAvailable_NoAccountID_ExitsWithOne(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		p := &pluginHintCmd{
			name:           "generate",
			description:    "Test description.",
			privatePreview: true,
			stdout:         os.Stdout,
			stdin:          strings.NewReader(""),
		}
		p.Command = &cobra.Command{Use: "generate", RunE: p.run}
		p.accountIDFn = func() (string, error) { return "", nil }
		p.suggestNotAvailable() //nolint:errcheck
		return
	}

	var stdout bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=TestSuggestNotAvailable_NoAccountID_ExitsWithOne")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	cmd.Stdout = &stdout

	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stdout.String(), "stripe login")
}

func TestSuggestNotAvailable_AccountIDError_ExitsWithOne(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		p := &pluginHintCmd{
			name:           "generate",
			description:    "Test description.",
			privatePreview: true,
			stdout:         os.Stdout,
			stdin:          strings.NewReader(""),
		}
		p.Command = &cobra.Command{Use: "generate", RunE: p.run}
		p.accountIDFn = func() (string, error) { return "", errors.New("not configured") }
		p.suggestNotAvailable() //nolint:errcheck
		return
	}

	var stdout bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=TestSuggestNotAvailable_AccountIDError_ExitsWithOne")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	cmd.Stdout = &stdout

	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stdout.String(), "stripe login")
}

func TestSuggestNotAvailable_ShowsAccountID_ExitsWithOne(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS") == "1" {
		p := &pluginHintCmd{
			name:           "generate",
			description:    "Test description.",
			privatePreview: true,
			stdout:         os.Stdout,
			stdin:          strings.NewReader(""),
		}
		p.Command = &cobra.Command{Use: "generate", RunE: p.run}
		p.accountIDFn = func() (string, error) { return "acct_abc456", nil }
		p.suggestNotAvailable() //nolint:errcheck
		return
	}

	var stdout bytes.Buffer
	cmd := exec.Command(os.Args[0], "-test.run=TestSuggestNotAvailable_ShowsAccountID_ExitsWithOne")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS=1")
	cmd.Stdout = &stdout

	err := cmd.Run()

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 1, exitErr.ExitCode())
	assert.Contains(t, stdout.String(), "acct_abc456")
}
