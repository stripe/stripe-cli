package cmd

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/stripe/stripe-cli/pkg/config"
	"github.com/stripe/stripe-cli/pkg/stripe"
)

type pluginInvocationEvent struct {
	name     string
	value    string
	metadata stripe.CLIAnalyticsEventMetadata
}

type pluginInvocationTelemetryClient struct {
	started chan struct{}
	resume  chan struct{}
	events  chan pluginInvocationEvent
}

func (*pluginInvocationTelemetryClient) SendAPIRequestEvent(context.Context, string, bool) (*http.Response, error) {
	return nil, nil
}

func (c *pluginInvocationTelemetryClient) SendEvent(ctx context.Context, name, value string) {
	close(c.started)
	<-c.resume
	c.events <- pluginInvocationEvent{name: name, value: value, metadata: *stripe.GetEventMetadata(ctx)}
}

func TestPluginInvocationTelemetryTarget(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantPlugin    string
		errorContains string
		wantInvoked   bool
	}{
		{name: "install latest", args: []string{"install", "apps"}, wantPlugin: "apps", wantInvoked: true},
		{name: "install exact version", args: []string{"install", "apps@2.0.0"}, wantPlugin: "apps", wantInvoked: true},
		{name: "install version never becomes identity", args: []string{"install", "apps@sk_test_private"}, wantPlugin: "apps", wantInvoked: true},
		{name: "upgrade", args: []string{"upgrade", "apps"}, wantPlugin: "apps", wantInvoked: true},
		{name: "install invalid target", args: []string{"install", "../private@2.0.0"}, errorContains: "invalid plugin name", wantInvoked: true},
		{name: "upgrade invalid target", args: []string{"upgrade", "../private"}, errorContains: "invalid plugin name", wantInvoked: true},
		{name: "install without target", args: []string{"install"}, errorContains: "requires exactly 1 positional argument"},
		{name: "upgrade extra target", args: []string{"upgrade", "apps", "private"}, errorContains: "requires exactly 1 positional argument"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			previousConfig, previousAccessBase := Config, rootAccessBaseURL
			t.Cleanup(func() {
				Config, rootAccessBaseURL = previousConfig, previousAccessBase
			})
			configRoot := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configRoot)
			Config = config.Config{
				LogLevel:     "info",
				ProfilesFile: filepath.Join(configRoot, "stripe", "config.toml"),
				Profile:      config.Profile{ProfileName: "default"},
			}
			rootAccessBaseURL = "https://access.stripe.com"
			preRunCalled := false
			root := &cobra.Command{
				Use:           "stripe",
				SilenceErrors: true,
				SilenceUsage:  true,
				PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
					preRunCalled = true
					return rootCmd.PersistentPreRunE(cmd, args)
				},
			}
			management := newPluginCmd()
			root.AddCommand(management.cmd)
			client := &pluginInvocationTelemetryClient{
				started: make(chan struct{}),
				resume:  make(chan struct{}),
				events:  make(chan pluginInvocationEvent, 1),
			}
			metadata := stripe.NewEventMetadata()
			ctx := stripe.WithEventMetadata(stripe.WithTelemetryClient(t.Context(), client), metadata)
			if test.wantInvoked && test.errorContains == "" {
				command, _, err := root.Find(append([]string{"plugin"}, test.args...))
				require.NoError(t, err)
				command.RunE = func(*cobra.Command, []string) error {
					<-client.started
					metadata.SetPluginName("later-target")
					metadata.SetPluginVersion("later-version")
					return nil
				}
			}
			root.SetArgs(append([]string{"plugin"}, test.args...))
			err := root.ExecuteContext(ctx)
			if test.errorContains == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.errorContains)
			}
			if !test.wantInvoked {
				require.False(t, preRunCalled, "invalid argument count reached invocation telemetry")
				return
			}
			require.True(t, preRunCalled)
			close(client.resume)
			event := <-client.events
			require.Equal(t, "Command Invoked", event.name)
			require.Equal(t, "Cobra", event.value)
			require.Equal(t, test.wantPlugin, event.metadata.PluginName)
			require.Empty(t, event.metadata.PluginVersion)
		})
	}
}
