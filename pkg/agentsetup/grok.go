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
	ClientGrok      = "grok"
	GrokBinaryName  = "grok"
	GrokPluginName  = "stripe"
	GrokDisplayName = "Grok"

	grokListTimeout = 5 * time.Second
)

// GrokProvider detects and installs the Stripe plugin for Grok Build (xAI).
type GrokProvider struct {
	ProviderConfig
}

// NewGrokProvider returns a Grok Build setup provider.
func NewGrokProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return GrokProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      ClientGrok,
			BinaryName:  GrokBinaryName,
			DisplayName: GrokDisplayName,
			RunCommand:  runCommand,
			RunOutput:   runCommandOutput,
		},
	}
}

func (p GrokProvider) ID() string { return p.Client }

func (p GrokProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusMissing,
	)
	if !status.Detected {
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), grokListTimeout)
	defer cancel()

	plugin, ok, supportsPlugins := p.stripePluginStatus(ctx)
	if !supportsPlugins {
		status.Error = "upgrade Grok Build to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.ID = plugin.Name
		status.Plugin.Version = plugin.Version
		status.Plugin.StatePath = plugin.Path
		status.Status = StatusInstalled
	}

	return status
}

// stripePluginStatus runs `grok plugin list --json` and reports whether the
// Stripe plugin is installed. When the command fails (e.g. an old Grok
// version without plugin support), supportsPlugins is false.
func (p GrokProvider) stripePluginStatus(ctx context.Context) (plugin grokInstalledPlugin, installed bool, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, p.BinaryName, "plugin", "list", "--json")
	if err != nil {
		return grokInstalledPlugin{}, false, false
	}
	plugin, ok := findGrokStripePlugin(out)
	return plugin, ok, true
}

// grokInstalledPlugin is an entry in `grok plugin list --json` output
type grokInstalledPlugin struct {
	Status      string `json:"status"`
	Name        string `json:"name"`
	RepoKey     string `json:"repo_key"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	Marketplace string `json:"marketplace"`
}

// findGrokStripePlugin reports whether the Stripe plugin appears in the
// output of `grok plugin list --json`.
func findGrokStripePlugin(listJSON []byte) (grokInstalledPlugin, bool) {
	var plugins []grokInstalledPlugin
	if err := json.Unmarshal(listJSON, &plugins); err != nil {
		return grokInstalledPlugin{}, false
	}

	for _, plugin := range plugins {
		if grokPluginIsStripe(plugin) {
			return plugin, true
		}
	}
	return grokInstalledPlugin{}, false
}

func grokPluginIsStripe(plugin grokInstalledPlugin) bool {
	return strings.EqualFold(plugin.Name, GrokPluginName) && strings.EqualFold(plugin.Status, "installed")
}

func (p GrokProvider) Plan(status Status, force bool) Plan {
	installCommand := [][]string{{p.BinaryName, "plugin", "install", GrokPluginName, "--trust"}}
	// `grok plugin install` is idempotent when already installed — it prints
	// "Plugin stripe is already installed ... Run `grok plugin update stripe`
	// to update it" rather than reinstalling, so a forced refresh has to go
	// through `update` instead.
	reinstallCommand := [][]string{{p.BinaryName, "plugin", "update", GrokPluginName}}
	return getPlanByStatus(status, force, installCommand, reinstallCommand)
}

// Apply installs (or updates) the Stripe Grok plugin. Unlike Codex's
// `plugin add` (see codex.go), `grok plugin install`/`update` exit non-zero
// on failure and 0 on success, so the exit code can be trusted without a
// post-install re-check.
func (p GrokProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}
	command := plan.Commands[0]
	return p.RunCommand(ctx, command[0], command[1:]...)
}
