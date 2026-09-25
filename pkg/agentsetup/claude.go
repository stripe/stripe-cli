// Package agentsetup contains helpers for detecting and configuring AI coding
// agent integrations.
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
	ClientClaudeCode   = "claude-code"
	ClaudeBinaryName   = "claude"
	TargetClaudePlugin = "stripe@claude-plugins-official"
	ClaudeMarketplace  = "claude-plugins-official"
	ClaudeDisplayName  = "Claude Code"

	claudeListTimeout = 5 * time.Second
)

// ClaudeProvider detects and configures the Stripe plugin for Claude Code.
type ClaudeProvider struct {
	ProviderConfig
}

// NewClaudeProvider returns a Claude Code setup provider.
func NewClaudeProvider(scanner Scanner, runCommand RunCommandFunc) Provider {
	if runCommand == nil {
		runCommand = RunCommand
	}
	return ClaudeProvider{
		ProviderConfig: ProviderConfig{
			Scanner:     scanner,
			Client:      ClientClaudeCode,
			BinaryName:  ClaudeBinaryName,
			DisplayName: ClaudeDisplayName,
			RunCommand:  runCommand,
			RunOutput:   runCommandOutput,
		},
	}
}

func (p ClaudeProvider) ID() string {
	return p.Client
}

func (p ClaudeProvider) Detect() Status {
	status := detectAgentExecutable(
		p.ProviderConfig,
		StatusMissing,
	)
	if !status.Detected {
		return status
	}

	ctx, cancel := context.WithTimeout(context.Background(), claudeListTimeout)
	defer cancel()

	pluginID, version, scope, ok, supportsPlugins := p.stripePluginStatus(ctx)
	if !supportsPlugins {
		status.Error = "upgrade Claude Code to enable plugin support"
		return status
	}
	if ok {
		status.Plugin.Installed = true
		status.Plugin.ID = pluginID
		status.Plugin.Version = version
		status.Plugin.Scope = scope
		status.Status = StatusInstalled
	}

	return status
}

func (p ClaudeProvider) Plan(status Status, force bool) Plan {
	name, args := p.installCommand()
	installOrReinstallCommand := [][]string{append([]string{name}, args...)}
	return getPlanByStatus(status, force, installOrReinstallCommand, installOrReinstallCommand)
}

// Apply installs the Stripe Claude Code plugin. On failure it silently refreshes
// the official marketplace and retries once.
func (p ClaudeProvider) Apply(ctx context.Context, _ io.Writer, plan Plan) error {
	if plan.Action == ActionNone {
		return nil
	}
	if len(plan.Commands) == 0 {
		return errorcategory.Errorf(errorcategory.Internal, "missing command for %s action", plan.Action)
	}

	command = plan.Commands[0]
	name, installArgs := command[0], command[1:]
	if err := p.RunCommand(ctx, name, installArgs...); err == nil {
		return nil
	}

	updateName, updateArgs := p.marketplaceUpdateCommand()
	if updateErr := p.RunCommand(ctx, updateName, updateArgs...); updateErr != nil {
		return updateErr
	}
	return p.RunCommand(ctx, name, installArgs...)
}

// stripePluginStatus runs `claude plugin list --json` and reports whether the
// Stripe plugin is installed. When the command fails (e.g. old Claude version
// without plugin support), supportsPlugins is false.
func (p ClaudeProvider) stripePluginStatus(ctx context.Context) (id, version, scope string, installed bool, supportsPlugins bool) {
	runOutput := p.RunOutput
	if runOutput == nil {
		runOutput = runCommandOutput
	}
	out, err := runOutput(ctx, p.BinaryName, "plugin", "list", "--json")
	if err != nil {
		return "", "", "", false, false
	}
	pluginID, v, s, ok := findClaudeStripePlugin(out)
	return pluginID, v, s, ok, true
}

// claudeInstalledPlugin is an entry in `claude plugin list --json` output.
type claudeInstalledPlugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Scope   string `json:"scope"`
	Enabled bool   `json:"enabled"`
}

// findClaudeStripePlugin reports whether the Stripe plugin appears in the
// output of `claude plugin list --json`.
func findClaudeStripePlugin(listJSON []byte) (id, version, scope string, found bool) {
	var plugins []claudeInstalledPlugin
	if err := json.Unmarshal(listJSON, &plugins); err != nil {
		return "", "", "", false
	}

	for _, plugin := range plugins {
		if claudePluginIsStripe(plugin) {
			return plugin.ID, plugin.Version, plugin.Scope, true
		}
	}
	return "", "", "", false
}

func claudePluginIsStripe(plugin claudeInstalledPlugin) bool {
	return strings.EqualFold(plugin.ID, TargetClaudePlugin)
}

// installCommand returns the command used to install the Stripe Claude Code
// plugin.
func (p ClaudeProvider) installCommand() (string, []string) {
	return p.BinaryName, []string{"plugin", "install", TargetClaudePlugin}
}

// marketplaceUpdateCommand returns the command used to refresh Claude's
// official plugin marketplace metadata.
func (p ClaudeProvider) marketplaceUpdateCommand() (string, []string) {
	return p.BinaryName, []string{"plugin", "marketplace", "update", ClaudeMarketplace}
}
