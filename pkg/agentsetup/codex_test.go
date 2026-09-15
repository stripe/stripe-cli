package agentsetup

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanCodex_NotDetected(t *testing.T) {
	provider := CodexProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }},
		RunOutput: func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("plugin list should not run when Codex is not detected")
			return nil, nil
		},
	}

	status := provider.Detect()

	require.Equal(t, ClientCodex, status.Client)
	require.False(t, status.Detected)
	require.Equal(t, StatusNotDetected, status.Status)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestScanCodex_PluginMissing(t *testing.T) {
	provider := codexTestProvider(`{"installed":[],"available":[]}`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", TargetCodexPlugin}}, provider.Plan(status, false))
}

func TestScanCodex_PluginInstalled(t *testing.T) {
	for _, marketplace := range []string{"openai-curated", "openai-api-curated"} {
		for _, tc := range []struct {
			name   string
			fields string
		}{
			{"full", `"pluginId":"stripe@%[1]s","name":"stripe","marketplaceName":"%[1]s"`},
			{"id_only", `"pluginId":"stripe@%s"`},
			{"name_and_marketplace", `"name":"stripe","marketplaceName":"%s"`},
		} {
			t.Run(marketplace+"/"+tc.name, func(t *testing.T) {
				listJSON := `{"installed":[{` + fmt.Sprintf(tc.fields, marketplace) + `,"version":"3fdeeb49"}]}`
				provider := codexTestProvider(listJSON, nil, nil)

				status := provider.Detect()

				pluginID := "stripe@" + marketplace
				require.Equal(t, StatusInstalled, status.Status)
				require.True(t, status.Plugin.Installed)
				require.Equal(t, pluginID, status.Plugin.ID)
				require.Equal(t, "3fdeeb49", status.Plugin.Version)
				provider.RunOutput = func(context.Context, string, ...string) ([]byte, error) {
					t.Fatal("an installed plugin should not need marketplace discovery")
					return nil, nil
				}
				require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
				require.Equal(t, Plan{Action: ActionReinstall, Command: []string{"codex", "plugin", "add", pluginID}}, provider.Plan(status, true))
			})
		}
	}
}

func TestScanCodex_OtherPluginsIgnored(t *testing.T) {
	for _, listJSON := range []string{
		`{"installed":[{"pluginId":"stripe@other-marketplace","name":"stripe","marketplaceName":"other-marketplace"}]}`,
		`{"installed":[{"pluginId":"other@openai-api-curated","name":"other","marketplaceName":"openai-api-curated"}]}`,
	} {
		provider := codexTestProvider(listJSON, nil, nil)
		status := provider.Detect()
		require.Equal(t, StatusMissing, status.Status)
		require.False(t, status.Plugin.Installed)
	}
}

func TestCodexPlan_Marketplace(t *testing.T) {
	for _, tc := range []struct {
		name       string
		output     string
		err        error
		wantPlugin string
	}{
		{
			name:       "curated",
			output:     `{"marketplaces":[{"name":"openai-curated","root":"/tmp/plugins"}]}`,
			wantPlugin: "stripe@openai-curated",
		},
		{
			name:       "api_curated",
			output:     `{"marketplaces":[{"name":"other-marketplace"},{"name":"openai-api-curated","root":"/tmp/plugins"}]}`,
			wantPlugin: "stripe@openai-api-curated",
		},
		{
			name:       "both_prefer_original_marketplace",
			output:     `{"marketplaces":[{"name":"openai-api-curated"},{"name":"openai-curated"}]}`,
			wantPlugin: "stripe@openai-curated",
		},
		{
			name:       "discovery_not_supported",
			err:        errors.New("unrecognized subcommand 'marketplace'"),
			wantPlugin: "stripe@openai-curated",
		},
		{
			name:       "invalid_json",
			output:     `{invalid`,
			wantPlugin: "stripe@openai-curated",
		},
		{
			name:       "no_marketplaces",
			output:     `{"marketplaces":[]}`,
			wantPlugin: "stripe@openai-curated",
		},
		{
			name:       "no_official_marketplace",
			output:     `{"marketplaces":[{"name":"other-marketplace"}]}`,
			wantPlugin: "stripe@openai-curated",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := codexTestProvider(`{"installed":[]}`, nil, nil)
			provider.RunOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
				require.Equal(t, "codex", name)
				switch strings.Join(args, " ") {
				case "plugin list --json":
					return []byte(`{"installed":[]}`), nil
				case "plugin marketplace list --json":
					return []byte(tc.output), tc.err
				default:
					t.Fatalf("unexpected command: %s %v", name, args)
					return nil, nil
				}
			}

			status := provider.Detect()

			require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", tc.wantPlugin}}, provider.Plan(status, false))
		})
	}
}

func TestScanCodex_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := codexTestProvider("", errors.New("unrecognized subcommand 'plugin'"), nil)

	status := provider.Detect()

	// Old Codex shows as detected but with an error hint — the TUI renders
	// it as disabled (visible but not selectable).
	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.Contains(t, status.Error, "upgrade Codex")
}

func TestCodexApply_RunsAddCommandAndVerifies(t *testing.T) {
	for _, marketplace := range []string{"openai-curated", "openai-api-curated"} {
		t.Run(marketplace, func(t *testing.T) {
			installed := false
			provider := CodexProvider{
				Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
				RunCommand: func(_ context.Context, name string, args ...string) error {
					require.Equal(t, "codex", name)
					require.Equal(t, []string{"plugin", "add", "stripe@" + marketplace}, args)
					installed = true
					return nil
				},
				RunOutput: func(_ context.Context, name string, args ...string) ([]byte, error) {
					switch strings.Join(args, " ") {
					case "plugin marketplace list --json":
						return []byte(fmt.Sprintf(`{"marketplaces":[{"name":%q}]}`, marketplace)), nil
					case "plugin list --json":
						if installed {
							return []byte(fmt.Sprintf(`{"installed":[{"pluginId":"stripe@%[1]s","name":"stripe","marketplaceName":"%[1]s","version":"1.0.0"}]}`, marketplace)), nil
						}
						return []byte(`{"installed":[]}`), nil
					default:
						t.Fatalf("unexpected command: %s %v", name, args)
						return nil, nil
					}
				},
			}

			status := provider.Detect()
			plan := provider.Plan(status, false)
			err := provider.Apply(context.Background(), nil, plan)

			require.NoError(t, err)
			require.True(t, installed)
		})
	}
}

// TestCodexApply_FailsWhenExitZeroButNotInstalled covers the real-world case
// where `codex plugin add` prints an error but exits 0. Apply must not report
// success when the plugin is still not present afterward.
func TestCodexApply_FailsWhenExitZeroButNotInstalled(t *testing.T) {
	for _, pluginID := range []string{"stripe@openai-curated", "stripe@openai-api-curated"} {
		t.Run(pluginID, func(t *testing.T) {
			provider := codexTestProvider(`{"installed":[]}`, nil, func(context.Context, string, ...string) error {
				return nil // add "succeeds" (exit 0) but installs nothing
			})
			plan := Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", pluginID}}
			err := provider.Apply(context.Background(), nil, plan)

			require.Error(t, err)
			require.Contains(t, err.Error(), pluginID+" is not installed")
			require.Contains(t, err.Error(), strings.Join(plan.Command, " "))
		})
	}
}

func codexTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) CodexProvider {
	return CodexProvider{
		Scanner:    Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: runCommand,
		RunOutput: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if strings.Join(args, " ") == "plugin marketplace list --json" {
				return []byte(`{"marketplaces":[{"name":"openai-curated"}]}`), nil
			}
			if listErr != nil {
				return nil, listErr
			}
			return []byte(listOutput), nil
		},
	}
}
