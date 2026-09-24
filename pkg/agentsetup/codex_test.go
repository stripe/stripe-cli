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
	scanner := Scanner{LookPath: func(string) (string, error) { return "", errors.New("missing") }}
	runOutput := func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("plugin list should not run when Codex is not detected")
		return nil, nil
	}
	provider := NewCodexProvider(scanner, nil).(CodexProvider)
	provider.RunOutput = runOutput

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
	require.Equal(t, "stripe@openai-curated", status.Plugin.ID)
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
	runOutput := provider.RunOutput
	provider.RunOutput = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if args[1] == "marketplace" {
			return []byte(`{"marketplaces":[{"name":"openai-api-curated"}]}`), nil
		}
		return runOutput(ctx, name, args...)
	}
	status := provider.Detect()

	require.Equal(t, StatusInstalled, status.Status)
	require.True(t, status.Plugin.Installed)
	require.Equal(t, "stripe@openai-api-curated", status.Plugin.ID)
	require.Equal(t, "1.0.0", status.Plugin.Version)
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, false))
	require.Equal(t, Plan{Action: ActionReinstall, Command: []string{"codex", "plugin", "add", "stripe@openai-api-curated"}}, provider.Plan(status, true))
}

func TestScanCodex_DoesNotFallBackAfterLookupError(t *testing.T) {
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

	require.Equal(t, []string{"openai-curated"}, marketplaces)
	require.Equal(t, StatusMissing, status.Status)
	require.Equal(t, "stripe@openai-curated", status.Plugin.ID)
	require.False(t, status.Plugin.Installed)
	require.Contains(t, status.Error, "upgrade Codex")
}

func TestScanCodex_OldVersionWithoutPluginSupport(t *testing.T) {
	provider := codexTestProvider("", errors.New("unrecognized subcommand 'plugin'"), nil)

	status := provider.Detect()

	require.True(t, status.Detected)
	require.Equal(t, StatusError, status.Status)
	require.Contains(t, status.Error, "listing Codex marketplaces")
	require.Equal(t, Plan{Action: ActionNone}, provider.Plan(status, true))
}

// An install failure, including exit zero without installing, must not cause
// an attempt from a different marketplace, even when both are available.
func TestCodexApply_DoesNotFallBack(t *testing.T) {
	for _, installErr := range []error{nil, errors.New("marketplace unavailable")} {
		var attempts []string
		provider := codexTestProvider(`{"installed":[]}`, nil, func(_ context.Context, _ string, args ...string) error {
			attempts = append(attempts, args[2])
			return installErr
		})

		status := provider.Detect()
		plan := provider.Plan(status, false)
		err := provider.Apply(context.Background(), nil, plan)

		require.Error(t, err)
		require.Equal(t, []string{"stripe@openai-curated"}, attempts)
		require.Contains(t, err.Error(), "openai-curated")
		require.NotContains(t, err.Error(), "openai-api-curated")
		if installErr != nil {
			require.ErrorIs(t, err, installErr)
		} else {
			require.ErrorContains(t, err, "codex reported success but stripe@openai-curated is not installed")
		}
	}
}

func TestCodexSetup_AvailableMarketplaces(t *testing.T) {
	for _, tt := range []struct {
		name        string
		listOutput  string
		marketplace string
	}{
		{
			name:        "supported order, ignoring unrelated and duplicate entries",
			listOutput:  `{"marketplaces":[{"name":"other"},{"name":"openai-api-curated"},{"name":"openai-curated"},{"name":"openai-curated"}]}`,
			marketplace: "openai-curated",
		},
		{
			name:        "ChatGPT marketplace only",
			listOutput:  `{"marketplaces":[{"name":"openai-curated"}]}`,
			marketplace: "openai-curated",
		},
		{
			name:        "API marketplace only",
			listOutput:  `{"marketplaces":[{"name":"openai-api-curated"}]}`,
			marketplace: "openai-api-curated",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var queried, attempts []string
			discoveries := 0
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
					discoveries++
					return []byte(tt.listOutput), nil
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
			require.Equal(t, []string{tt.marketplace}, queried)
			pluginID := "stripe@" + tt.marketplace
			require.Equal(t, pluginID, status.Plugin.ID)
			plan := provider.Plan(status, false)
			require.Equal(t, Plan{Action: ActionInstall, Command: []string{"codex", "plugin", "add", pluginID}}, plan)
			require.NoError(t, provider.Apply(context.Background(), nil, plan))
			require.Equal(t, []string{pluginID}, attempts)
			require.Equal(t, 1, discoveries, "Plan and Apply must reuse the selection from Detect")

			status = provider.Detect()
			require.Equal(t, StatusInstalled, status.Status)
			require.Equal(t, pluginID, status.Plugin.ID)
			require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, false)))
			require.Equal(t, []string{pluginID}, attempts, "skip an installed plugin")
			require.NoError(t, provider.Apply(context.Background(), nil, provider.Plan(status, true)))
			require.Equal(t, []string{pluginID, pluginID}, attempts, "force reinstalls from the detected marketplace")
			require.Equal(t, 2, discoveries, "reinstall must also reuse the selection from Detect")
		})
	}
}

func TestScanCodex_MarketplaceDiscoveryFailures(t *testing.T) {
	for _, tt := range []struct {
		name       string
		listOutput string
		listErr    error
		wantError  string
	}{
		{
			name:       "no marketplaces",
			listOutput: `{"marketplaces":[]}`,
			wantError:  "no supported Codex marketplace is available",
		},
		{
			name:       "only unrelated marketplaces",
			listOutput: `{"marketplaces":[{"name":"other"}]}`,
			wantError:  "no supported Codex marketplace is available",
		},
		{
			name:      "listing fails",
			listErr:   errors.New("marketplace listing failed"),
			wantError: "listing Codex marketplaces: marketplace listing failed",
		},
		{
			name:      "listing times out",
			listErr:   context.DeadlineExceeded,
			wantError: "listing Codex marketplaces: context deadline exceeded",
		},
		{
			name:       "invalid JSON",
			listOutput: "not JSON",
			wantError:  "listing Codex marketplaces: invalid character",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			provider := codexTestProvider("", nil, func(context.Context, string, ...string) error {
				t.Fatal("must not install without a detected marketplace")
				return nil
			})
			discoveries := 0
			provider.RunOutput = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				require.Equal(t, []string{"plugin", "marketplace", "list", "--json"}, args)
				discoveries++
				return []byte(tt.listOutput), tt.listErr
			}

			status := provider.Detect()
			require.Equal(t, StatusError, status.Status)
			require.Contains(t, status.Error, tt.wantError)
			require.Empty(t, status.Plugin.ID)
			for _, force := range []bool{false, true} {
				plan := provider.Plan(status, force)
				require.Equal(t, Plan{Action: ActionNone}, plan)
				require.NoError(t, provider.Apply(context.Background(), nil, plan))
			}
			require.Equal(t, 1, discoveries)
		})
	}
}

func codexTestProvider(listOutput string, listErr error, runCommand RunCommandFunc) CodexProvider {
	scanner := Scanner{LookPath: func(string) (string, error) { return "/usr/local/bin/codex", nil }}
	runOutput := func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if listErr != nil {
			return nil, listErr
		}
		if args[1] == "marketplace" {
			return []byte(codexTestMarketplaceList), nil
		}
		return []byte(listOutput), nil
	}
	provider := NewCodexProvider(scanner, runCommand).(CodexProvider)
	provider.RunOutput = runOutput
	return provider
}
