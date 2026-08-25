package pluginhints

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/login"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

// pluginFound and pluginAutoInstallable stand in for the metadata lookup. The
// resolved version carries the server's answer to whether the auto-install
// rollout has reached this machine, so a test picks which side of that rollout it
// is running on by picking between these two.
func pluginFound(context.Context) (*plugins.ResolvedPluginVersion, error) {
	return &plugins.ResolvedPluginVersion{}, nil
}

func pluginAutoInstallable(context.Context) (*plugins.ResolvedPluginVersion, error) {
	return &plugins.ResolvedPluginVersion{AutoInstall: true}, nil
}

// pluginLookupFails stands in for a lookup that could not reach an answer at all.
func pluginLookupFails(message string) func(context.Context) (*plugins.ResolvedPluginVersion, error) {
	return func(context.Context) (*plugins.ResolvedPluginVersion, error) {
		return nil, errors.New(message)
	}
}

// newTestCmd builds a pluginHintCmd with all side effects mocked out.
// By default accountIDFn reports a logged-in account; override in tests that
// need to simulate an unauthenticated user.
func newTestCmd(name string, opts ...option) *pluginHintCmd {
	p := &pluginHintCmd{
		name:          name,
		description:   "Test description.",
		stdout:        &bytes.Buffer{},
		stderr:        &bytes.Buffer{},
		stdin:         strings.NewReader(""),
		accountIDFn:   func() (string, error) { return "acct_test", nil },
		loginFn:       func(ctx context.Context) error { return nil },
		accessBaseURL: login.DefaultAccessBaseURL,
		argvFn:        func() []string { return []string{"stripe", name} },
		lookupEnvFn:   func(string) string { return "" },
	}
	for _, opt := range opts {
		opt(p)
	}
	// Use the production command wiring so flag and help handling can't drift from
	// what a real invocation gets.
	p.initCommand()
	// The host normally points Cobra at stdout; do the same so help rendered by
	// Cobra lands in the same buffer as this command's own output.
	p.SetOut(p.stdout)
	return p
}

func (p *pluginHintCmd) output() string {
	return p.stdout.(*bytes.Buffer).String()
}

// errOutput returns what the command wrote to stderr, which is where the
// non-interactive auto-install path reports progress and next steps.
func (p *pluginHintCmd) errOutput() string {
	return p.stderr.(*bytes.Buffer).String()
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
	rootCmd := &cobra.Command{Use: "stripe", Annotations: map[string]string{}}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{}, nil)

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
	rootCmd := &cobra.Command{Use: "stripe", Annotations: map[string]string{}}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{
		"directory": true,
	}, nil)

	assert.Nil(t, findChildCommand(rootCmd, "directory"))
}

func TestAddHintCommands_SetsAvailablePluginAnnotations(t *testing.T) {
	rootCmd := &cobra.Command{Use: "stripe", Annotations: map[string]string{}}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{}, nil)

	for _, name := range []string{"apps", "generate", "projects", "directory", "tools"} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, "available_plugin", rootCmd.Annotations[name])
		})
	}
}

func TestAddHintCommands_NoAnnotationWhenPluginInstalled(t *testing.T) {
	rootCmd := &cobra.Command{Use: "stripe", Annotations: map[string]string{}}

	AddHintCommands(rootCmd, &config.Config{}, map[string]bool{
		"tools": true,
		"apps":  true,
	}, nil)

	assert.Empty(t, rootCmd.Annotations["tools"])
	assert.Empty(t, rootCmd.Annotations["apps"])
	assert.Equal(t, "available_plugin", rootCmd.Annotations["generate"])
	assert.Equal(t, "available_plugin", rootCmd.Annotations["projects"])
	assert.Equal(t, "available_plugin", rootCmd.Annotations["directory"])
}

// --- run ---

func TestRun_PluginFound_CallsPromptInstall(t *testing.T) {
	p := newTestCmd("generate", withPrivatePreview())
	installCalled := false
	p.lookupFn = pluginFound
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.output(), "The \"generate\" plugin is required")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_LoggedIn_PrintsInstallHint(t *testing.T) {
	p := newTestCmd("apps") // accountIDFn returns "acct_test" by default
	p.lookupFn = pluginLookupFails("not found")

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Contains(t, p.output(), "stripe plugin install apps")
}

func TestRun_PluginNotFound_PrivatePreviewFalse_NotLoggedIn_PromptsLogin(t *testing.T) {
	p := newTestCmd("docs")
	p.lookupFn = pluginLookupFails("not found")
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
	p.lookupFn = pluginLookupFails("not found")
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
			accessBaseURL:  login.DefaultAccessBaseURL,
		}
		p.Command = &cobra.Command{Use: "generate", RunE: p.run}
		p.lookupFn = pluginLookupFails("not found")
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

// --- auto-install ---

// runViaCobra executes p through a root command so Cobra resolves aliases and
// records CalledAs the way it does during a real invocation. argv stands in for
// os.Args, so it includes the binary name.
func runViaCobra(t *testing.T, p *pluginHintCmd, aliases []string, argv []string) error {
	t.Helper()

	p.argvFn = func() []string { return argv }
	p.Aliases = aliases
	p.SilenceUsage = true
	p.SilenceErrors = true

	root := &cobra.Command{Use: "stripe", SilenceUsage: true, SilenceErrors: true}
	// Stand in for the host's persistent flags, which Cobra consumes before RunE.
	root.PersistentFlags().String("log-level", "", "")
	root.AddCommand(p.Command)
	root.SetArgs(argv[1:])
	root.SetOut(p.stdout)
	root.SetErr(p.stdout)

	return root.Execute()
}

// newAutoInstallTestCmd builds a command whose metadata lookup reports that the
// auto-install rollout has reached this machine, which is what makes the
// auto-install paths below reachable at all.
func newAutoInstallTestCmd(name string, opts ...option) *pluginHintCmd {
	p := newTestCmd(name, opts...)
	p.lookupFn = pluginAutoInstallable
	p.installFn = func(ctx context.Context) error { return nil }
	return p
}

func TestRun_AutoInstall_InstallsAndRunsWithoutPrompting(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
	var ranWith []string
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { ranWith = args; return nil }
	// A prompt would consume this; auto-install must leave stdin for the plugin.
	stdin := strings.NewReader("unread\n")
	p.stdin = stdin

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.NotNil(t, ranWith, "expected the plugin to run after installing")
	assert.Contains(t, p.errOutput(), "one-time setup")
	assert.NotContains(t, p.output()+p.errOutput(), "press Enter")
	remaining, readErr := io.ReadAll(stdin)
	require.NoError(t, readErr)
	assert.Equal(t, "unread\n", string(remaining))
}

func TestRun_AutoInstall_ForwardsArgsToPlugin(t *testing.T) {
	tests := []struct {
		name     string
		aliases  []string
		argv     []string
		wantArgs []string
	}{
		{
			name:     "canonical name",
			argv:     []string{"stripe", "directory", "search", "coffee shops"},
			wantArgs: []string{"search", "coffee shops"},
		},
		{
			name:     "host flags before the plugin name are not forwarded",
			argv:     []string{"stripe", "--log-level", "debug", "directory", "search", "x"},
			wantArgs: []string{"search", "x"},
		},
		{
			name:     "plugin flags after the plugin name are forwarded",
			argv:     []string{"stripe", "directory", "search", "x", "--limit", "5"},
			wantArgs: []string{"search", "x", "--limit", "5"},
		},
		{
			name:     "bare invocation forwards no args",
			argv:     []string{"stripe", "directory"},
			wantArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			var ranWith []string
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { ranWith = args; return nil }

			err := runViaCobra(t, p, tt.aliases, tt.argv)

			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, ranWith)
		})
	}
}

// TestRun_AutoInstall_AliasPromptsInsteadOfInstalling pins the rule that fetching a
// binary without asking requires the plugin's real name. An alias — a typo alias
// especially — is too weak a signal to act on silently, so it falls back to the same
// prompt every other not-yet-installed plugin uses.
func TestRun_AutoInstall_AliasPromptsInsteadOfInstalling(t *testing.T) {
	tests := []struct {
		name        string
		aliases     []string
		argv        []string
		wantInstall bool
	}{
		{
			name:        "the real name installs without asking",
			argv:        []string{"stripe", "directory", "search", "coffee shops"},
			wantInstall: true,
		},
		{
			name:    "subcommand alias asks first",
			aliases: []string{"search"},
			argv:    []string{"stripe", "search", "coffee shops"},
		},
		{
			name:    "typo alias asks first",
			aliases: []string{"direcotry"},                          //nolint:misspell // Intentional typo alias.
			argv:    []string{"stripe", "direcotry", "search", "x"}, //nolint:misspell // Intentional typo alias.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			// Declining at the prompt is how this test tells the two paths apart:
			// auto-install never reads stdin.
			p.stdin = strings.NewReader("no\n")
			installed := false
			p.installFn = func(ctx context.Context) error { installed = true; return nil }

			err := runViaCobra(t, p, tt.aliases, tt.argv)

			if tt.wantInstall {
				require.NoError(t, err)
				assert.True(t, installed)
				assert.NotContains(t, p.output(), "press Enter")
				return
			}

			assert.EqualError(t, err, "installation canceled")
			assert.False(t, installed, "an alias must not install without asking")
			assert.Contains(t, p.output(), "press Enter to install")
		})
	}
}

func TestRun_AutoInstall_InstallErrorSkipsPluginRun(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.installFn = func(ctx context.Context) error { return errors.New("install failed") }
	runCalled := false
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { runCalled = true; return nil }

	err := p.run(p.Command, nil)

	assert.EqualError(t, err, "install failed")
	assert.False(t, runCalled)
}

func TestRun_AutoInstall_PropagatesPluginError(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { return errors.New("plugin blew up") }

	err := p.run(p.Command, nil)

	assert.EqualError(t, err, "plugin blew up")
}

func TestRun_AutoInstall_PrintsNextStepsBeforeRunningPlugin(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	var errOutputAtRunTime string
	p.runPluginFn = func(cmd *cobra.Command, args []string) error {
		errOutputAtRunTime = p.errOutput()
		return nil
	}

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	// Captured before the plugin produced any output, so the tips cannot be
	// mistaken for part of the command's result.
	assert.Contains(t, errOutputAtRunTime, "installation complete")
	assert.Contains(t, errOutputAtRunTime, "stripe directory search")
	assert.Contains(t, errOutputAtRunTime, "directory@stripe.com")
}

// TestRun_AutoInstall_LeavesStdoutToThePlugin protects the first-run experience for
// anything parsing the output: installing on demand must not prepend human-readable
// setup chatter to the stream the plugin's own result arrives on.
func TestRun_AutoInstall_LeavesStdoutToThePlugin(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	var stdoutAtRunTime string
	p.runPluginFn = func(cmd *cobra.Command, args []string) error {
		stdoutAtRunTime = p.output()
		fmt.Fprintln(p.stdout, `{"results":[]}`)
		return nil
	}

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Empty(t, stdoutAtRunTime, "stdout must be untouched when the plugin takes over")
	assert.Equal(t, "{\"results\":[]}\n", p.output())
	// The guidance still has to reach the user, just not on stdout.
	assert.Contains(t, p.errOutput(), "installation complete")
}

func TestRun_AutoInstall_NoRunnerPrintsNextSteps(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.True(t, installCalled)
	assert.Contains(t, p.errOutput(), "installation complete")
	assert.Contains(t, p.errOutput(), "directory@stripe.com")
}

func TestRun_AutoInstall_OptedOutDoesNotInstallOrPrompt(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE"} {
		t.Run(value, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			p.lookupEnvFn = func(key string) string {
				if key == AutoInstallOptOutEnvVar {
					return value
				}
				return ""
			}
			lookupCalled := false
			p.lookupFn = func(ctx context.Context) (*plugins.ResolvedPluginVersion, error) {
				lookupCalled = true
				return nil, errors.New("lookup unavailable")
			}
			p.accountIDFn = func() (string, error) { return "", nil }
			installCalled := false
			p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
			runCalled := false
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { runCalled = true; return nil }
			loginCalled := false
			p.loginFn = func(ctx context.Context) error { loginCalled = true; return nil }
			stdin := strings.NewReader("unread\n")
			p.stdin = stdin

			err := p.run(p.Command, nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "stripe plugin install directory")
			assert.Contains(t, err.Error(), AutoInstallOptOutEnvVar)
			assert.False(t, lookupCalled)
			assert.False(t, installCalled)
			assert.False(t, runCalled)
			assert.False(t, loginCalled)
			// The host prints the error, so repeating it on either stream would say
			// the same thing twice.
			assert.Empty(t, p.output())
			assert.Empty(t, p.errOutput())
			remaining, readErr := io.ReadAll(stdin)
			require.NoError(t, readErr)
			assert.Equal(t, "unread\n", string(remaining))
		})
	}
}

func TestRun_AutoInstall_UnsetOrFalsyOptOutStillInstalls(t *testing.T) {
	for _, value := range []string{"", "0", "false", "nonsense"} {
		t.Run(value, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			p.lookupEnvFn = func(string) string { return value }
			installCalled := false
			p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { return nil }

			err := p.run(p.Command, nil)

			require.NoError(t, err)
			assert.True(t, installCalled)
		})
	}
}

func TestRun_AutoInstall_LookupFailureStillFallsBackToHints(t *testing.T) {
	// Auto-install must not mask the not-available and login paths.
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.lookupFn = pluginLookupFails("not found")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.Contains(t, p.output(), "stripe plugin install directory")
}

// TestRun_AutoInstall_ServerDecisionGatesInstalling pins that opting a plugin into
// auto-install only makes it eligible: the metadata endpoint decides per machine,
// so a machine the rollout has not reached gets the same prompt as every other
// not-yet-installed plugin.
func TestRun_AutoInstall_ServerDecisionGatesInstalling(t *testing.T) {
	tests := []struct {
		name        string
		lookupFn    func(context.Context) (*plugins.ResolvedPluginVersion, error)
		wantInstall bool
	}{
		{
			name:        "rollout reached this machine",
			lookupFn:    pluginAutoInstallable,
			wantInstall: true,
		},
		{
			name:     "rollout has not reached this machine",
			lookupFn: pluginFound,
		},
		{
			// An older server sends no auto_install field at all, which decodes to the
			// prompting behavior the CLI has always had.
			name: "server did not answer",
			lookupFn: func(context.Context) (*plugins.ResolvedPluginVersion, error) {
				return &plugins.ResolvedPluginVersion{Version: "1.2.3"}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			p.lookupFn = tt.lookupFn
			// Declining at the prompt is how this test tells the two paths apart:
			// auto-install never reads stdin.
			p.stdin = strings.NewReader("no\n")
			installed := false
			p.installFn = func(ctx context.Context) error { installed = true; return nil }
			ran := false
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { ran = true; return nil }

			err := runViaCobra(t, p, nil, []string{"stripe", "directory", "search", "x"})

			if tt.wantInstall {
				require.NoError(t, err)
				assert.True(t, installed)
				assert.True(t, ran)
				assert.NotContains(t, p.output(), "press Enter")
				return
			}

			assert.EqualError(t, err, "installation canceled")
			assert.False(t, installed, "the server did not opt this machine in")
			assert.False(t, ran)
			assert.Contains(t, p.output(), "press Enter to install")
		})
	}
}

func TestRun_WithoutAutoInstall_StillPrompts(t *testing.T) {
	p := newAutoInstallTestCmd("apps")
	p.stdin = strings.NewReader("\n")

	err := p.run(p.Command, nil)

	require.NoError(t, err)
	assert.Contains(t, p.output(), "press Enter to install")
}

// --- auto-install help ---

func TestHelp_AutoInstall_InstallsAndForwardsHelpToPlugin(t *testing.T) {
	tests := []struct {
		name     string
		aliases  []string
		argv     []string
		wantArgs []string
	}{
		{
			name:     "help flag",
			argv:     []string{"stripe", "directory", "--help"},
			wantArgs: []string{"--help"},
		},
		{
			name:     "short help flag is forwarded as typed",
			argv:     []string{"stripe", "directory", "-h"},
			wantArgs: []string{"-h"},
		},
		{
			name:     "help flag on a plugin subcommand",
			argv:     []string{"stripe", "directory", "search", "--help"},
			wantArgs: []string{"search", "--help"},
		},
		{
			name: "help subcommand",
			argv: []string{"stripe", "help", "directory"},
			// argv carries no help flag for the plugin to act on, so one is added.
			wantArgs: []string{"--help"},
		},
		{
			name:     "help subcommand with a plugin subcommand",
			argv:     []string{"stripe", "help", "directory", "search"},
			wantArgs: []string{"search", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			installCalled := false
			p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
			var ranWith []string
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { ranWith = args; return nil }

			err := runViaCobra(t, p, tt.aliases, tt.argv)

			require.NoError(t, err)
			assert.True(t, installCalled)
			assert.Equal(t, tt.wantArgs, ranWith)
			// The plugin supplies the help text, so the placeholder's must not appear.
			assert.NotContains(t, p.output(), "Test description.")
			assert.Contains(t, p.errOutput(), "one-time setup")
			assert.Contains(t, p.errOutput(), "directory@stripe.com")
		})
	}
}

// TestHelp_AutoInstall_AliasShowsPlaceholderHelpWithoutInstalling is the help-side
// half of the rule in TestRun_AutoInstall_AliasPromptsInsteadOfInstalling: asking an
// alias what it does must not download anything.
func TestHelp_AutoInstall_AliasShowsPlaceholderHelpWithoutInstalling(t *testing.T) {
	tests := []struct {
		name string
		argv []string
	}{
		{name: "help flag on an alias", argv: []string{"stripe", "search", "--help"}},
		{name: "help subcommand on an alias", argv: []string{"stripe", "help", "search"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
			installCalled := false
			p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
			p.runPluginFn = func(cmd *cobra.Command, args []string) error { return nil }

			err := runViaCobra(t, p, []string{"search"}, tt.argv)

			require.NoError(t, err)
			assert.False(t, installCalled, "help on an alias must not install the plugin")
			assert.Contains(t, p.output(), "Test description.", "expected the placeholder help")
			assert.Empty(t, p.errOutput())
		})
	}
}

func TestHelp_AutoInstall_OptedOutShowsPlaceholderHelpWithoutInstalling(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.lookupEnvFn = func(string) string { return "1" }
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
	runCalled := false
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { runCalled = true; return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "directory", "--help"})

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.False(t, runCalled)
	assert.Contains(t, p.output(), "Test description.")
	assert.Contains(t, p.output(), AutoInstallOptOutEnvVar)
	assert.Contains(t, p.output(), "stripe plugin install directory")
	// The env var name is the whole reason here, so it must not be repeated once the
	// caller has added the install command.
	assert.Equal(t, 1, strings.Count(p.output(), "stripe plugin install directory"))
}

func TestHelp_AutoInstall_LookupFailureShowsPlaceholderHelp(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.lookupFn = pluginLookupFails("no metadata")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "directory", "--help"})

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.Contains(t, p.output(), "Test description.")
	assert.Contains(t, p.output(), "no metadata")
}

func TestHelp_AutoInstall_InstallFailureShowsPlaceholderHelp(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.installFn = func(ctx context.Context) error { return errors.New("install failed") }
	runCalled := false
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { runCalled = true; return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "directory", "--help"})

	// Cobra's help hook cannot fail, so the user gets the placeholder plus the reason
	// the plugin's own help is missing.
	require.NoError(t, err)
	assert.False(t, runCalled)
	// Both halves have to survive: the help the user asked for, and why it is only
	// the placeholder.
	assert.Contains(t, p.output(), "Test description.")
	assert.Contains(t, p.output(), "Usage:")
	assert.Contains(t, p.output(), "install failed")
	assert.Contains(t, p.output(), "stripe plugin install directory")
	// The reason has to come after the help text, so it is the last thing on screen
	// rather than scrolled off above it.
	assert.Less(t, strings.Index(p.output(), "Usage:"), strings.Index(p.output(), "install failed"))
}

func TestHelp_AutoInstall_NoRunnerShowsPlaceholderHelpWithoutInstalling(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "directory", "--help"})

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.Contains(t, p.output(), "Test description.")
	// Nothing was attempted, so there is no gap to explain.
	assert.NotContains(t, p.output(), "unavailable")
}

// TestHelp_AutoInstall_ServerOptOutShowsPlaceholderHelpQuietly is the help-side half
// of TestRun_AutoInstall_ServerDecisionGatesInstalling. The rollout is not the user's
// business, so a machine it has not reached sees the plain placeholder help with no
// explanation of why the plugin did not document itself.
func TestHelp_AutoInstall_ServerOptOutShowsPlaceholderHelpQuietly(t *testing.T) {
	p := newAutoInstallTestCmd("directory", withAutoInstall(nil))
	p.lookupFn = pluginFound
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }
	runCalled := false
	p.runPluginFn = func(cmd *cobra.Command, args []string) error { runCalled = true; return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "directory", "--help"})

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.False(t, runCalled)
	assert.Contains(t, p.output(), "Test description.")
	assert.NotContains(t, p.output(), "unavailable")
	assert.NotContains(t, p.output(), "stripe plugin install directory")
	assert.Empty(t, p.errOutput())
}

func TestHelp_WithoutAutoInstall_ShowsPlaceholderHelp(t *testing.T) {
	p := newAutoInstallTestCmd("apps")
	installCalled := false
	p.installFn = func(ctx context.Context) error { installCalled = true; return nil }

	err := runViaCobra(t, p, nil, []string{"stripe", "apps", "--help"})

	require.NoError(t, err)
	assert.False(t, installCalled)
	assert.Contains(t, p.output(), "Test description.")
	assert.NotContains(t, p.output()+p.errOutput(), "one-time setup")
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

func TestPromptInstall_Directory_PrintsNextSteps(t *testing.T) {
	p := newTestCmd("directory")
	p.stdin = strings.NewReader("\n")
	p.installFn = func(ctx context.Context) error { return nil }
	err := p.promptInstall(context.Background())
	require.NoError(t, err)
	assert.Contains(t, p.output(), "directory@stripe.com")
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
