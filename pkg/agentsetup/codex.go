package agentsetup

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/errorcategory"
)

const (
	ClientCodex      = "codex"
	CodexBinaryName  = "codex"
	CodexPluginName  = "stripe"
	CodexDisplayName = "Codex CLI"

	codexListTimeout = 5 * time.Second
)

// Official Codex marketplaces for ChatGPT and API-key users, respectively:
// https://github.com/openai/plugins/blob/main/.agents/plugins/marketplace.json#L2
// https://github.com/openai/plugins/blob/main/.agents/plugins/api_marketplace.json
var codexMarketplaces = [...]string{"openai-curated", "openai-api-curated"}

// RunOutputFunc runs a command and returns its standard output. It exists so
// Codex detection (which shells out to `codex plugin list --json`) is testable.
type RunOutputFunc func(context.Context, string, ...string) ([]byte, error)

// CodexProvider detects and installs the Stripe plugin for Codex CLI.
//
// Detection selects the first available supported marketplace, and installation
// runs `codex plugin add stripe@<marketplace>` using that selection.
type CodexProvider struct {
	ProviderConfig
}

// NewCodexProvider returns a Codex CLI setup provider.
func NewCodexProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}

	return CodexProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      ClientCodex,
			BinaryName:  CodexBinaryName,
			DisplayName: CodexDisplayName,
			RunCommand:  runCommand,
			RunOutput:   runCommandOutput,
		},
	}
}

func (p CodexProvider) ID() string { return p.Client }

func (p CodexProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusMissing,
	)

	if !status.Detected {
		return status
	}

	marketplace, err := p.marketplace(context.Background())
	if err != nil {
		status.Status = StatusError
		status.Error = err.Error()
		return status
	}
	status.Plugin.ID = CodexPluginName + "@" + marketplace

	ctx, cancel := context.WithTimeout(context.Background(), codexListTimeout)
	defer cancel()

	version, ok, supportsPlugins := p.stripePluginStatus(ctx, marketplace)
	if !supportsPlugins {
		status.Error = "upgrade Codex to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.Version = version
		status.Plugin.Scope = "user"
		status.Status = StatusInstalled
	}

	return status
}

// marketplace selects the first available supported marketplace.
func (p CodexProvider) marketplace(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, codexListTimeout)
	defer cancel()
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, p.BinaryName, "plugin", "marketplace", "list", "--json")
	var list struct {
		Marketplaces []struct {
			Name string `json:"name"`
		} `json:"marketplaces"`
	}
	if err == nil {
		err = json.Unmarshal(out, &list)
	}
	if err != nil {
		return "", errorcategory.Errorf(errorcategory.Internal, "listing Codex marketplaces: %w", err)
	}

	for _, marketplace := range codexMarketplaces {
		for _, available := range list.Marketplaces {
			if available.Name == marketplace {
				return marketplace, nil
			}
		}
	}
	return "", errorcategory.Errorf(errorcategory.Internal, "no supported Codex marketplace is available; expected %s", strings.Join(codexMarketplaces[:], " or "))
}

// stripePluginStatus runs `codex plugin list --json` and reports whether (1)
// the command is supported (supportsPlugins), and if so (2) whether the Stripe
// plugin is installed and its version. When the command fails (e.g. old Codex
// version without plugin support), supportsPlugins is false.
func (p CodexProvider) stripePluginStatus(ctx context.Context, marketplace string) (version string, installed bool, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	// The unfiltered list can omit locally installed curated plugins.
	out, err := runOutput(ctx, p.BinaryName, "plugin", "list", "--marketplace", marketplace, "--json")
	if err != nil {
		return "", false, false
	}
	v, ok := findCodexStripePlugin(out, marketplace)
	return v, ok, true
}

func (p CodexProvider) Plan(status Status, force bool) Plan {
	installOrReinstallCommand := [][]string{{p.BinaryName, "plugin", "add", status.Plugin.ID}}
	return getPlanByStatus(status, force, installOrReinstallCommand, installOrReinstallCommand)
}

func (p CodexProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}
	runCommand := p.RunCommand
	if runCommand == nil {
		runCommand = RunCommand
	}
	command := plan.Commands[0]
	pluginID := command[len(command)-1]
	_, marketplace, _ := strings.Cut(pluginID, "@")
	if err := runCommand(ctx, command[0], command[1:]...); err != nil {
		return errorcategory.Errorf(errorcategory.Internal, "could not install the Stripe plugin from %s: %w", marketplace, err)
	}

	// `codex plugin add` exits 0 even when it fails (e.g. the marketplace is not
	// configured), so the exit code cannot be trusted. Confirm the plugin is
	// actually installed before reporting success.
	if _, installed, _ := p.stripePluginStatus(ctx, marketplace); !installed {
		return errorcategory.Errorf(errorcategory.Internal, "codex reported success but %s is not installed; run `%s` to see the underlying error",
			pluginID, strings.Join(command, " "))
	}
	return nil
}

// codexPluginList is the shape of `codex plugin list --json` output.
type codexPluginList struct {
	Installed []codexInstalledPlugin `json:"installed"`
}

// codexInstalledPlugin is an entry in `codex plugin list --json`'s "installed"
// array. Field names match the real Codex CLI output (verified against
// codex-cli 0.142.0), e.g. {"pluginId":"stripe@openai-curated","name":"stripe",
// "marketplaceName":"openai-curated","version":"..."}.
type codexInstalledPlugin struct {
	PluginID    string `json:"pluginId"`
	Name        string `json:"name"`
	Marketplace string `json:"marketplaceName"`
	Version     string `json:"version"`
}

// findCodexStripePlugin reports whether the Stripe plugin appears in the
// installed list and returns its version when available.
func findCodexStripePlugin(listJSON []byte, marketplace string) (string, bool) {
	var list codexPluginList
	if err := json.Unmarshal(listJSON, &list); err != nil {
		return "", false
	}

	for _, plugin := range list.Installed {
		if codexPluginIsStripe(plugin, marketplace) {
			return plugin.Version, true
		}
	}
	return "", false
}

func codexPluginIsStripe(plugin codexInstalledPlugin, marketplace string) bool {
	if strings.EqualFold(plugin.PluginID, CodexPluginName+"@"+marketplace) {
		return true
	}
	return strings.EqualFold(plugin.Name, CodexPluginName) &&
		strings.EqualFold(plugin.Marketplace, marketplace)
}
