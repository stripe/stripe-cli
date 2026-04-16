package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/plugins"
)

func createPluginCmd(cfg *config.Config) *pluginTemplateCmd {
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

	pluginCmd := newPluginTemplateCmd(cfg, &plugin)

	return pluginCmd
}

// TestFlagsArePassedAsArgs ensures that the plugin is passing all args and flags as expected.
// This is a complex dance between the CLI itself and the plugin, so the flags come from
// two different sources as a result. This test is here to catch any non-obvious regressions
func TestFlagsArePassedAsArgs(t *testing.T) {
	cfg := &config.Config{}
	pluginCmd := createPluginCmd(cfg)
	rootCmd.AddCommand(pluginCmd.cmd)

	Execute(context.Background())

	// temp override for the os.Args so that the pluginCmd can use them
	oldArgs := os.Args
	os.Args = []string{"stripe", "test", "testarg", "--log-level=info"}
	defer func() { os.Args = oldArgs }()

	rootCmd.SetArgs([]string{"test", "testarg", "--log-level=info"})
	executeCommandC(rootCmd, "test", "testarg", "--log-level=info")

	require.Equal(t, 2, len(pluginCmd.ParsedArgs))
	require.Equal(t, "testarg --log-level=info", strings.Join(pluginCmd.ParsedArgs, " "))
}

func TestAddPluginSubcommandStubs(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "myapp",
		Shortdesc:        "My app plugin",
		Binary:           "stripe-cli-myapp",
		MagicCookieValue: "magic",
		Commands: []plugins.CommandInfo{
			{
				Name: "create",
				Desc: "Create a resource",
			},
			{
				Name: "logs",
				Desc: "View logs",
				Commands: []plugins.CommandInfo{
					{
						Name: "tail",
						Desc: "Tail logs in real-time",
					},
				},
			},
		},
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// Verify subcommand stubs were created
	subCmds := ptc.cmd.Commands()
	require.Equal(t, 2, len(subCmds))

	assert.Equal(t, "create", subCmds[0].Name())
	assert.Equal(t, "Create a resource", subCmds[0].Short)
	assert.Equal(t, "plugin", subCmds[0].Annotations["scope"])

	assert.Equal(t, "logs", subCmds[1].Name())
	assert.Equal(t, "View logs", subCmds[1].Short)

	// Verify nested subcommand
	logSubCmds := subCmds[1].Commands()
	require.Equal(t, 1, len(logSubCmds))
	assert.Equal(t, "tail", logSubCmds[0].Name())
	assert.Equal(t, "Tail logs in real-time", logSubCmds[0].Short)
	assert.Equal(t, "plugin", logSubCmds[0].Annotations["scope"])
}

func TestAddPluginSubcommandStubsEmpty(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "simple",
		Shortdesc:        "A simple plugin",
		Binary:           "stripe-cli-simple",
		MagicCookieValue: "magic",
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// No subcommands should be created
	assert.Equal(t, 0, len(ptc.cmd.Commands()))
}

func TestAddPluginSubcommandStubsSkipsEmptyName(t *testing.T) {
	plugin := plugins.Plugin{
		Shortname:        "badplugin",
		Shortdesc:        "A plugin with bad manifest data",
		Binary:           "stripe-cli-bad",
		MagicCookieValue: "magic",
		Commands: []plugins.CommandInfo{
			{Name: "valid", Desc: "A valid command"},
			{Name: "", Desc: "Entry with empty name"},
			{Name: "also-valid", Desc: "Another valid command"},
		},
	}

	ptc := newPluginTemplateCmd(&Config, &plugin)

	// Only the two valid entries should become subcommands
	cmds := ptc.cmd.Commands()
	assert.Equal(t, 2, len(cmds))
	assert.Equal(t, "also-valid", cmds[0].Name())
	assert.Equal(t, "valid", cmds[1].Name())
}

// TestWithBackgroundUpdate_FnRuns verifies that fn is invoked and completes.
func TestWithBackgroundUpdate_FnRuns(t *testing.T) {
	cfg := &config.Config{}
	fs := afero.NewMemMapFs()
	plugin := plugins.Plugin{Shortname: "test"}

	called := false
	err := plugins.WithBackgroundUpdate(context.Background(), cfg, fs, "", &plugin, io.Discard, func() error {
		called = true
		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)
}

// TestWithBackgroundUpdate_PropagatesError verifies that an error returned by fn
// is returned by WithBackgroundUpdate.
func TestWithBackgroundUpdate_PropagatesError(t *testing.T) {
	cfg := &config.Config{}
	fs := afero.NewMemMapFs()
	plugin := plugins.Plugin{Shortname: "test"}

	want := errors.New("plugin failed")
	err := plugins.WithBackgroundUpdate(context.Background(), cfg, fs, "", &plugin, io.Discard, func() error {
		return want
	})

	assert.Equal(t, want, err)
}

// TestWithBackgroundUpdate_UpdateOutputAppearsAfterFn verifies that any output
// written by the background update goroutine is only flushed to the underlying
// writer after fn returns — never interleaved with fn's execution.
func TestWithBackgroundUpdate_UpdateOutputAppearsAfterFn(t *testing.T) {
	cfg := &config.Config{}
	fs := afero.NewMemMapFs()
	plugin := plugins.Plugin{Shortname: "test"}

	var mu sync.Mutex
	var events []string

	// recordWriter appends each Write as an event.
	out := &funcWriter{fn: func(p []byte) (int, error) {
		mu.Lock()
		events = append(events, "write:"+string(p))
		mu.Unlock()
		return len(p), nil
	}}

	err := plugins.WithBackgroundUpdate(context.Background(), cfg, fs, "", &plugin, out, func() error {
		mu.Lock()
		events = append(events, "fn:done")
		mu.Unlock()
		return nil
	})

	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	// If there were any writes (update output), they must come after fn:done.
	fnIdx := -1
	for i, e := range events {
		if e == "fn:done" {
			fnIdx = i
		}
	}
	// fn must have run
	require.GreaterOrEqual(t, fnIdx, 0, "fn:done not recorded")
	for i, e := range events {
		if strings.HasPrefix(e, "write:") {
			assert.Greater(t, i, fnIdx, "update write at index %d appeared before fn:done at index %d", i, fnIdx)
		}
	}
}

// TestWithBackgroundUpdate_NilErrorOnSuccess verifies a nil error on clean fn exit.
func TestWithBackgroundUpdate_NilErrorOnSuccess(t *testing.T) {
	cfg := &config.Config{}
	fs := afero.NewMemMapFs()
	plugin := plugins.Plugin{Shortname: "test"}

	// Provide a non-nil out to ensure the writer path is exercised.
	var buf bytes.Buffer
	err := plugins.WithBackgroundUpdate(context.Background(), cfg, fs, "", &plugin, &buf, func() error {
		return nil
	})

	assert.NoError(t, err)
}

// funcWriter is a minimal io.Writer backed by a function, used in tests.
type funcWriter struct {
	fn func([]byte) (int, error)
}

func (f *funcWriter) Write(p []byte) (int, error) { return f.fn(p) }

func TestSubsliceAfter(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
		sl       []string
		str      string
	}{
		{"empty slice", []string{}, []string{}, "foo"},
		{"empty string", []string{}, []string{""}, ""},
		{"not found", []string{}, []string{"bar"}, "foo"},
		{"found at beginning", []string{"bar"}, []string{"foo", "bar"}, "foo"},
		{"found at middle", []string{"baz", "qux"}, []string{"foo", "bar", "baz", "qux"}, "bar"},
		{"found at end", []string{}, []string{"foo", "bar"}, "bar"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, subsliceAfter(tt.sl, tt.str))
		})
	}
}
