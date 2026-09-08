package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsHelpRequest(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{"no args", []string{}, false},
		{"unrelated args", []string{"neon/postgres", "--config", "{}"}, false},
		{"short help flag", []string{"-h"}, true},
		{"long help flag", []string{"--help"}, true},
		{"help flag among other args", []string{"neon/postgres", "--help"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isHelpRequest(tt.args))
		})
	}
}

func TestBuildProvisionPluginArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{"no args", []string{}, []string{"add"}},
		{
			"service and flags",
			[]string{"neon/postgres", "--config", `{"region":"iad1"}`, "--json"},
			[]string{"add", "neon/postgres", "--config", `{"region":"iad1"}`, "--json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildProvisionPluginArgs(tt.args))
		})
	}
}

// TestProvisionHelpShortCircuitsBeforeInvokingPlugin ensures `stripe provision --help`
// renders cobra's own help output and never attempts to look up or run the projects
// plugin (which would fail/hang in a test environment without a real plugin installed).
func TestProvisionHelpShortCircuitsBeforeInvokingPlugin(t *testing.T) {
	pc := newProvisionCmd()

	invoked := false
	pc.runProvisionCmdFn = func(cmd *cobra.Command, args []string) error {
		invoked = true
		return pc.runProvisionCmd(cmd, args)
	}

	_, output, err := executeCommandC(pc.cmd, "--help")

	require.NoError(t, err)
	assert.True(t, invoked, "runProvisionCmdFn should still be called; the short-circuit happens inside it")
	assert.Contains(t, output, "alias for 'stripe projects add'")
	assert.Contains(t, output, "Examples:")
}

// TestProvisionCmdForwardsRawArgsUnchanged mirrors TestFlagsArePassedAsArgs for the
// plugin template command: with DisableFlagParsing set, cobra must hand the raw,
// unparsed args straight to the RunE func so they can be forwarded to the plugin.
func TestProvisionCmdForwardsRawArgsUnchanged(t *testing.T) {
	pc := newProvisionCmd()

	var capturedArgs []string
	pc.runProvisionCmdFn = func(cmd *cobra.Command, args []string) error {
		capturedArgs = append([]string(nil), args...)
		return nil
	}

	_, _, err := executeCommandC(pc.cmd, "neon/postgres", "--config", `{"region":"iad1"}`, "--json")

	require.NoError(t, err)
	require.Equal(t, []string{"neon/postgres", "--config", `{"region":"iad1"}`, "--json"}, capturedArgs)
}

func TestProvisionCmdUsage(t *testing.T) {
	pc := newProvisionCmd()

	assert.Equal(t, "provision [service]", pc.cmd.Use)
	assert.True(t, strings.Contains(pc.cmd.Long, "stripe projects add"))
}
