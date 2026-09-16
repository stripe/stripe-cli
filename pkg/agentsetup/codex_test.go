package agentsetup

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const codexTestMarketplaceList = `{"marketplaces":[{"name":"openai-curated"},{"name":"openai-api-curated"}]}`

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
}

func TestScanCodex_PluginMissing(t *testing.T) {
	provider := codexTestProvider(`{"installed":[],"available":[]}`, nil, nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusMissing, status.Status)
	require.False(t, status.Plugin.Installed)
	require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", "stripe@openai-curated"}}, provider.Plan(status, false))
}

func TestScanCodex_PluginInstalled(t *testing.T) {
	// Real codex-cli schema: pluginId + marketplaceName.
	provider := codexTestProvider(`{"installed":[{"pluginId":"stripe@openai-curated","name":"stripe","marketplaceName":"openai-curated","version":"3fdeeb49"}]}`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe@openai-curated", status.Plugin.ID)
	require.Equal(t, "3fdeeb49", status.Plugin.Version)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
}

func TestScanCodex_PluginInstalledByNameAndMarketplace(t *testing.T) {
	// Entry without pluginId still matches on name + marketplaceName.
	provider := codexTestProvider(`{"installed":[{"name":"stripe","marketplaceName":"openai-curated","version":"2.0.0"}]}`, nil, nil)

	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.Equal(t, "2.0.0", status.Plugin.Version)
}

func TestScanCodex_APIPluginInstalled(t *testing.T) {
	provider := codexTestProvider(`{"installed":[{"pluginId":"stripe@openai-api-curated","version":"1.0.0"}]}`, nil, nil)
	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe@openai-api-curated", status.Plugin.ID)
	require.Equal(t, "1.0.0", status.Plugin.Version)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t, Plan{Action: ActionReinstall, Command: []string{"codex", "plugin", "add", "stripe@openai-curated"}}, provider.Plan(status, true))
}

func TestScanCodex_FallsBackAfterMarketplaceError(t *testing.T) {
	provider := codexTestProvider("", nil, nil)
	var marketplaces []string
	provider.RunOutput = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "marketplace" {
			return []byte(codexTestMarketplaceList), nil
		}
		marketplace := args[3]
		marketplaces = append(marketplaces, marketplace)
		if marketplace == "openai-curated" {
			return nil, errors.New("marketplace unavailable")
		}
		return []byte(`{"installed":[{"pluginId":"stripe@openai-api-curated","version":"1.0.0"}]}`), nil
	}

	status := provider.Detect()

	require.Equal(t, []string{"openai-curated", "openai-api-curated"}, marketplaces)
	require.Equal(t, StatusInstalled, status.Status)
	require.Equal(t, "stripe@openai-api-curated", status.Plugin.ID)
	require.Equal(t, "1.0.0", status.Plugin.Version)
	require.Empty(t, status.Error)
}

func TestCodexApply_APIMarketplace(t *testing.T) {
	// Codex may fail with a nonzero exit or exit zero without installing anything.
	for _, installErr := range []error{nil, errors.New("marketplace unavailable")} {
		installed := false
		var attempts []string
		provider := codexTestProvider("", nil, func(_ context.Context, name string, args ...string) error {
			require.Equal(t, "codex", name)
			require.Equal(t, []string{"plugin", "add"}, args[:2])
			attempts = append(attempts, args[2])
			if args[2] == "stripe@openai-curated" {
				return installErr
			}
			require.Equal(t, "stripe@openai-api-curated", args[2])
			installed = true
			return nil
		})
		provider.RunOutput = func(_ context.Context, name string, args ...string) ([]byte, error) {
			require.Equal(t, "codex", name)
			if args[1] == "marketplace" {
				return []byte(codexTestMarketplaceList), nil
			}
			require.Equal(t, []string{"plugin", "list", "--marketplace"}, args[:3])
			require.Equal(t, "--json", args[4])
			if installed && args[3] == "openai-api-curated" {
				return []byte(`{"installed":[{"name":"stripe","marketplaceName":"openai-api-curated","version":"1.0.0"}]}`), nil
			}
			return []byte(`{"installed":[]}`), nil
		}

		plan := provider.Plan(provider.Detect(), false)
		require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", "stripe@openai-curated"}}, plan)
		require.NoError(t, provider.Apply(context.Background(), nil, plan))
		require.Equal(t, []string{"stripe@openai-curated", "stripe@openai-api-curated"}, attempts)

		attempts = nil
		require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(provider.Detect(), true)))
		require.Equal(t, []string{"stripe@openai-curated", "stripe@openai-api-curated"}, attempts)
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
	var gotName string
	var gotArgs []string
	installed := false

	provider := CodexProvider{
		Scanner: Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: func(_ context.Context, name string, args ...string) error {
			require.False(t, installed, "should stop after the first successful install")
			gotName = name
			gotArgs = args
			installed = true // simulate a successful add
			return nil
		},
		RunOutput: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if args[1] == "marketplace" {
				return []byte(codexTestMarketplaceList), nil
			}
			if installed {
				return []byte(`{"installed":[{"pluginId":"stripe@openai-curated","name":"stripe","marketplaceName":"openai-curated","version":"1.0.0"}]}`), nil
			}
			return []byte(`{"installed":[]}`), nil
		},
	}

	status := provider.Detect()
	plan := provider.Plan(status, false)
	err := provider.Apply(context.Background(), nil, plan)

	require.NoError(t, err)
	require.Equal(t, "codex", gotName)
	require.Equal(t, []string{"plugin", "add", "stripe@openai-curated"}, gotArgs)

	status.Plugin = PluginStatus{Installed: true, ID: "stripe@openai-api-curated"}
	installed = false
	require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, true)))
	require.Equal(t, []string{"plugin", "add", "stripe@openai-curated"}, gotArgs)
}

// TestCodexApply_FailsWhenExitZeroButNotInstalled covers the real-world case
// where `codex plugin add` prints an error but exits 0. Apply must not report
// success when the plugin is still not present afterward.
func TestCodexApply_FailsWhenExitZeroButNotInstalled(t *testing.T) {
	for _, installErr := range []error{nil, errors.New("marketplace unavailable")} {
		provider := codexTestProvider(`{"installed":[]}`, nil, func(_ context.Context, _ string, args ...string) error {
			if args[2] == "stripe@openai-api-curated" {
				return installErr
			}
			return nil // add "succeeds" (exit 0) but installs nothing
		})

		status := provider.Detect()
		plan := provider.Plan(status, false)
		err := provider.Apply(context.Background(), nil, plan)

		require.Error(t, err)
		require.Contains(t, err.Error(), "is not installed")
		require.Contains(t, err.Error(), "openai-curated")
		require.Contains(t, err.Error(), "openai-api-curated")
		if installErr != nil {
			require.ErrorIs(t, err, installErr)
		}
	}
}

func TestCodexSetup_AvailableMarketplaces(t *testing.T) {
	for _, tt := range []struct {
		name         string
		listOutput   string
		listErr      error
		marketplaces []string
	}{
		{
			name:         "supported order, ignoring unrelated and duplicate entries",
			listOutput:   `{"marketplaces":[{"name":"other"},{"name":"openai-api-curated"},{"name":"openai-curated"},{"name":"openai-curated"}]}`,
			marketplaces: []string{"openai-curated", "openai-api-curated"},
		},
		{
			name:         "ChatGPT marketplace only",
			listOutput:   `{"marketplaces":[{"name":"openai-curated"}]}`,
			marketplaces: []string{"openai-curated"},
		},
		{
			name:         "API marketplace only",
			listOutput:   `{"marketplaces":[{"name":"openai-api-curated"}]}`,
			marketplaces: []string{"openai-api-curated"},
		},
		{
			name:         "listing fails",
			listErr:      context.DeadlineExceeded,
			marketplaces: []string{"openai-curated", "openai-api-curated"},
		},
		{
			name:         "invalid marketplace JSON",
			listOutput:   "not JSON",
			marketplaces: []string{"openai-curated", "openai-api-curated"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var queried, attempts []string
			installed := ""
			provider := codexTestProvider("", nil, func(_ context.Context, name string, args ...string) error {
				require.Equal(t, "codex", name)
				require.Equal(t, []string{"plugin", "add"}, args[:2])
				attempts = append(attempts, args[2])
				installed = args[2]
				return nil
			})
			provider.RunOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
				require.Equal(t, "codex", name)
				require.NoError(t, ctx.Err())
				if args[1] == "marketplace" {
					require.Equal(t, []string{"plugin", "marketplace", "list", "--json"}, args)
					return []byte(tt.listOutput), tt.listErr
				}
				require.Equal(t, []string{"plugin", "list", "--marketplace"}, args[:3])
				require.Equal(t, "--json", args[4])
				queried = append(queried, args[3])
				if installed == "stripe@"+args[3] {
					return []byte(fmt.Sprintf(`{"installed":[{"pluginId":%q,"version":"1.0.0"}]}`, installed)), nil
				}
				return []byte(`{"installed":[]}`), nil
			}

			status := provider.Detect()
			require.Equal(t, StatusMissing, status.Status)
			require.Empty(t, status.Error)
			require.Equal(t, tt.marketplaces, queried)
			require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, false)))
			first := "stripe@" + tt.marketplaces[0]
			require.Equal(t, []string{first}, attempts)

			status = provider.Detect()
			require.Equal(t, StatusInstalled, status.Status)
			require.Equal(t, first, status.Plugin.ID)
			require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, false)))
			require.Equal(t, []string{first}, attempts, "skip an installed plugin")
			require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, true)))
			require.Equal(t, []string{first, first}, attempts, "force reinstalls from the first selected marketplace")
		})
	}
}

func TestCodexApply_MarketplaceFailures(t *testing.T) {
	listErr := errors.New("marketplace listing failed")
	installErr := errors.New("plugin unavailable")
	for _, tt := range []struct {
		name       string
		listOutput string
		listErr    error
		attempts   []string
		reasons    []string
	}{
		{
			name:       "no marketplaces",
			listOutput: `{"marketplaces":[]}`,
			reasons:    []string{"openai-curated: marketplace is not available", "openai-api-curated: marketplace is not available"},
		},
		{
			name:       "only unrelated marketplaces",
			listOutput: `{"marketplaces":[{"name":"other"}]}`,
			reasons:    []string{"openai-curated: marketplace is not available", "openai-api-curated: marketplace is not available"},
		},
		{
			name:       "only available marketplace fails",
			listOutput: `{"marketplaces":[{"name":"openai-api-curated"}]}`,
			attempts:   []string{"stripe@openai-api-curated"},
			reasons:    []string{"openai-curated: marketplace is not available", "openai-api-curated: plugin unavailable"},
		},
		{
			name:     "listing and both installs fail",
			listErr:  listErr,
			attempts: []string{"stripe@openai-curated", "stripe@openai-api-curated"},
			reasons:  []string{"listing Codex marketplaces: marketplace listing failed", "openai-curated: plugin unavailable", "openai-api-curated: plugin unavailable"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var attempts []string
			provider := codexTestProvider("", nil, func(_ context.Context, _ string, args ...string) error {
				attempts = append(attempts, args[2])
				return installErr
			})
			provider.RunOutput = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				if args[1] == "marketplace" {
					return []byte(tt.listOutput), tt.listErr
				}
				return []byte(`{"installed":[]}`), nil
			}

			status := provider.Detect()
			require.Empty(t, status.Error, "unavailable marketplaces should not show an upgrade hint")
			err := provider.Apply(context.Background(), nil, provider.Plan(status, false))
			require.ErrorContains(t, err, "could not install the Stripe plugin from openai-curated or openai-api-curated")
			require.Equal(t, tt.attempts, attempts)
			for _, reason := range tt.reasons {
				require.ErrorContains(t, err, reason)
			}
			if tt.listErr != nil {
				require.ErrorIs(t, err, tt.listErr)
			}
			if len(attempts) > 0 {
				require.ErrorIs(t, err, installErr)
			}
		})
	}
}

func codexTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) CodexProvider {
	return CodexProvider{
		Scanner:    Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }},
		RunCommand: runCommand,
		RunOutput: func(_ context.Context, _ string, args ...string) ([]byte, error) {
			if listErr != nil {
				return nil, listErr
			}
			if args[1] == "marketplace" {
				return []byte(codexTestMarketplaceList), nil
			}
			return []byte(listOutput), nil
		},
	}
}
